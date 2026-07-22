package gpt

import (
	"cmp"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/xid"
)

const (
	MaxRetries               = 3
	MaxOutputTokens          = 2000
	MaxToolCallRounds        = 3
	DefaultSystemInstruction = "Du befindest dich in einer Telegram-Gruppenkonversation mit mehreren Nutzern. Nachrichten sind mit dem jeweiligen Nutzernamen vorangestellt. Antworte nur auf Deutsch. Markdown ist DEAKTIVIERT. HTML ist DEAKTIVIERT. Bilder-Analyse ist AKTIVIERT."
)

var (
	log        = logger.New("gpt")
	httpClient = httpUtils.NewHTTPClientWithTimeout(30 * time.Second)
)

type (
	Plugin struct {
		credentialService model.CredentialService
		gptService        Service
	}

	Service interface {
		GetResponseID(chat *gotgbot.Chat) (model.GPTData, error)
		ResetResponseID(chat *gotgbot.Chat) error
		SetResponseID(chat *gotgbot.Chat, responseID string) error
	}
)

func New(credentialService model.CredentialService, gptService Service) *Plugin {
	return &Plugin{
		credentialService: credentialService,
		gptService:        gptService,
	}
}

func (p *Plugin) Name() string {
	return "gpt"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)^Bot, ([\s\S]+)$`),
			HandlerFunc: p.onGPT,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/botreset(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onReset,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/botreset(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.onResetAndRun,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) makeRequest(req *Request, apiKey string) (Response, error) {
	var apiResponse Response
	var apiErr APIErrorResponse
	err := httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:        httpUtils.MethodPost,
		URL:           ApiURL,
		Headers:       map[string]string{"Authorization": fmt.Sprintf("Bearer %s", apiKey)},
		Body:          req,
		Response:      &apiResponse,
		ErrorResponse: &apiErr,
		Client:        httpClient,
	})
	if err != nil {
		if apiErr.Error.Message != "" {
			log.Err(err).
				Str("api_error_type", apiErr.Error.Type).
				Str("api_error_code", apiErr.Error.Code).
				Str("api_error_message", apiErr.Error.Message).
				Msg("OpenAI API error")
		}
		return apiResponse, err
	}
	return apiResponse, nil
}

func (p *Plugin) sendWithRetry(req *Request, apiKey string) (Response, error) {
	resp, err := p.makeRequest(req, apiKey)
	for retryCount := 0; retryCount < MaxRetries; retryCount++ {
		if err == nil {
			return resp, nil
		}
		httpError, ok := errors.AsType[*httpUtils.HttpError](err)
		if !ok || !isRetryableStatus(httpError.StatusCode) {
			return resp, err
		}
		wait := time.Duration(1<<retryCount) * time.Second
		log.Warn().
			Err(err).
			Int("status_code", httpError.StatusCode).
			Int("retry_count", retryCount).
			Dur("wait", wait).
			Msg("Received server error, retrying")
		time.Sleep(wait)
		resp, err = p.makeRequest(req, apiKey)
	}
	return resp, err
}

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func (p *Plugin) handleAPIError(b *gotgbot.Bot, c plugin.GobotContext, err error) error {
	if httpError, ok := errors.AsType[*httpUtils.HttpError](err); ok {
		if httpError.StatusCode == http.StatusBadRequest {
			guid := xid.New().String()
			log.Err(err).Str("guid", guid).Msg("HTTP 400, resetting response ID")
			if resetErr := p.gptService.ResetResponseID(c.EffectiveChat); resetErr != nil {
				log.Error().Err(resetErr).Int64("chat_id", c.EffectiveChat.Id).Msg("error resetting GPT data")
			}
			_, err := c.EffectiveMessage.ReplyMessage(b,
				fmt.Sprintf("❌ Es ist ein Fehler aufgetreten, Konversation wird zurückgesetzt.%s", utils.EmbedGUID(guid)),
				utils.DefaultSendOptions(),
			)
			return err
		}
		if httpError.StatusCode == http.StatusTooManyRequests {
			_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions())
			return err
		}
	}
	if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Timeout, bitte erneut versuchen.", utils.DefaultSendOptions())
		return err
	}
	guid := xid.New().String()
	log.Err(err).Str("guid", guid).Msg("Failed to send POST request")
	_, err = c.EffectiveMessage.ReplyMessage(b,
		fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
		utils.DefaultSendOptions(),
	)
	return err
}

func (p *Plugin) onGPT(b *gotgbot.Bot, c plugin.GobotContext) error {
	apiKey := p.credentialService.GetKey("openai_api_key")
	if apiKey == "" {
		log.Warn().Msg("openai_api_key not found")
		_, err := c.EffectiveMessage.ReplyMessage(b,
			"❌ <code>openai_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	systemInstruction := cmp.Or(p.credentialService.GetKey("openai_system_instruction"), DefaultSystemInstruction)
	systemInstruction += fmt.Sprintf("\n\nHeute ist %s.", utils.LocalizeDatestring(time.Now().Format("Monday, der 02.01.2006")))
	braveKey := p.credentialService.GetKey("brave_search_api_key")
	gptModel := cmp.Or(p.credentialService.GetKey("openai_model"), DefaultModel)

	registeredTools := []Tool{NewWebfetchTool(c.EffectiveChat.Id), NewCalculatorTool()}
	if braveKey != "" {
		registeredTools = append(registeredTools, NewWebsearchTool(braveKey, c.EffectiveChat.Id))
	}

	toolDefs := make([]FunctionTool, len(registeredTools))
	toolMap := make(map[string]Tool, len(registeredTools))
	for i, t := range registeredTools {
		def := t.Definition()
		toolDefs[i] = def
		toolMap[def.Name] = t
	}

	var previousResponseID string
	gptData, err := p.gptService.GetResponseID(c.EffectiveChat)
	if err != nil {
		log.Error().
			Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error getting GPT data")
	}
	if gptData.ResponseID.Valid && gptData.ExpiresOn.Valid {
		if time.Now().Before(gptData.ExpiresOn.Time) {
			previousResponseID = gptData.ResponseID.String
		}
	}

	var photo *gotgbot.PhotoSize
	var inputText strings.Builder

	if tgUtils.IsReply(c.EffectiveMessage) {
		photo = tgUtils.GetBestResolution(c.EffectiveMessage.ReplyToMessage.Photo)
		if c.EffectiveMessage.ReplyToMessage.GetText() != "" {
			inputText.WriteString("-- ZUSÄTZLICHER KONTEXT --\n")
			inputText.WriteString("Dies ist zusätzlicher Kontext. Wiederhole diesen nicht wortwörtlich!\n\n")
			inputText.WriteString("Nachricht")
			if from := c.EffectiveMessage.ReplyToMessage.From; from != nil {
				inputText.WriteString(fmt.Sprintf(" von %s", from.FirstName))
				if from.LastName != "" {
					inputText.WriteString(fmt.Sprintf(" %s", from.LastName))
				}
			}
			inputText.WriteString(":\n")
			inputText.WriteString(c.EffectiveMessage.ReplyToMessage.GetText())

			if c.EffectiveMessage.Quote != nil && c.EffectiveMessage.Quote.Text != "" {
				inputText.WriteString("\n-- Beziehe dich nur auf folgenden Textteil: --\n")
				inputText.WriteString(c.EffectiveMessage.Quote.Text)
			}

			inputText.WriteString("\n-- ZUSÄTZLICHER KONTEXT ENDE --\n")
		}
	}

	if c.EffectiveMessage.ExternalReply != nil && c.EffectiveMessage.ExternalReply.Photo != nil {
		photo = tgUtils.GetBestResolution(c.EffectiveMessage.ExternalReply.Photo)
	}

	if c.EffectiveMessage.Photo != nil {
		photo = tgUtils.GetBestResolution(c.EffectiveMessage.Photo)
	}

	inputText.WriteString(fmt.Sprintf("%s: %s", c.EffectiveMessage.From.FirstName, c.Matches[1]))

	content := []any{
		InputText{Type: TypeInputText, Text: inputText.String()},
	}

	if photo != nil {
		_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)

		if photo.FileSize > tgUtils.MaxFilesizeDownload {
			log.Warn().Msgf("File is too big: %d", photo.FileSize)
			_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Das Bild ist zu groß.", utils.DefaultSendOptions())
			return err
		}

		file, err := httpUtils.DownloadFile(b, photo.FileId)
		if err != nil {
			log.Err(err).
				Interface("photo", photo).
				Msg("Failed to get photo from Telegram")
			_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Konnte Bild nicht von Telegram herunterladen.", utils.DefaultSendOptions())
			return err
		}
		defer func(file io.ReadCloser) {
			if closeErr := file.Close(); closeErr != nil {
				log.Err(closeErr).Msg("Failed to close file")
			}
		}(file)

		imageBytes, err := io.ReadAll(file)
		if err != nil {
			guid := xid.New().String()
			log.Err(err).Str("guid", guid).Msg("Failed to read image bytes")
			_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		encoded := base64.StdEncoding.EncodeToString(imageBytes)
		content = append(content, InputImage{
			Type:     TypeInputImage,
			ImageURL: fmt.Sprintf("data:image/jpeg;base64,%s", encoded),
		})
	}

	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	req := &Request{
		Model: gptModel,
		Input: []any{
			InputMessage{Role: RoleUser, Content: content},
		},
		Instructions:       systemInstruction,
		Store:              true,
		MaxOutputTokens:    MaxOutputTokens,
		PreviousResponseID: previousResponseID,
		Tools:              toolDefs,
		Reasoning:          Reasoning{Effort: "none"},
	}

	apiResponse, err := p.sendWithRetry(req, apiKey)
	if err != nil {
		return p.handleAPIError(b, c, err)
	}

	usedToolNames := make(map[string]struct{})

	for round := range MaxToolCallRounds {
		var calls []OutputItem
		for _, item := range apiResponse.Output {
			if item.Type == TypeFunctionCall {
				calls = append(calls, item)
			}
		}
		if len(calls) == 0 {
			break
		}

		_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

		toolInputs := make([]any, len(calls))
		var mu sync.Mutex
		var wg sync.WaitGroup

		for i, call := range calls {
			wg.Add(1)
			go func(i int, call OutputItem) {
				defer wg.Done()
				// A panic here would kill the whole process; the handler-level
				// recover in bot/processor.go does not cover this goroutine.
				defer func() {
					if r := recover(); r != nil {
						log.Error().Interface("panic", r).Str("tool", call.Name).Msg("Tool panicked")
						toolInputs[i] = FunctionCallOutput{
							Type:   TypeFunctionCallOutput,
							CallID: call.CallID,
							Output: fmt.Sprintf("Error: tool panicked: %v", r),
						}
					}
				}()
				var toolOutput any
				if tool, ok := toolMap[call.Name]; !ok {
					toolOutput = fmt.Sprintf("Unknown tool: %s", call.Name)
				} else {
					result, execErr := tool.Execute(call.Arguments)
					if execErr != nil {
						toolOutput = fmt.Sprintf("Error: %v", execErr)
					} else {
						toolOutput = result
					}
					mu.Lock()
					usedToolNames[call.Name] = struct{}{}
					mu.Unlock()
				}
				toolInputs[i] = FunctionCallOutput{
					Type:   TypeFunctionCallOutput,
					CallID: call.CallID,
					Output: toolOutput,
				}
			}(i, call)
		}
		wg.Wait()

		toolReq := &Request{
			Model:              gptModel,
			Input:              toolInputs,
			Instructions:       systemInstruction,
			Store:              true,
			MaxOutputTokens:    MaxOutputTokens,
			PreviousResponseID: apiResponse.ID,
			Tools:              toolDefs,
			Reasoning:          Reasoning{Effort: "none"},
		}

		// In the final round, drop tools so the model is forced to produce a
		// text answer instead of requesting more tool calls we can't satisfy.
		if round == MaxToolCallRounds-1 {
			toolReq.Tools = nil
		}

		apiResponse, err = p.sendWithRetry(toolReq, apiKey)
		if err != nil {
			return p.handleAPIError(b, c, err)
		}
	}

	var outputText strings.Builder
	for _, out := range apiResponse.Output {
		if out.Type == TypeMessage {
			for _, part := range out.Content {
				if part.Type == TypeOutputText {
					outputText.WriteString(part.Text)
				}
			}
		}
	}

	output := outputText.String()

	if output == "" {
		log.Error().Str("status", apiResponse.Status).Msg("Got no answer from GPT")
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Keine Antwort von GPT erhalten (eventuell gefiltert).", utils.DefaultSendOptions())
		return err
	}

	if apiResponse.Status == StatusIncomplete {
		log.Warn().Msg("GPT response is incomplete")
		output += " […]"
	}

	if len(usedToolNames) > 0 {
		var prefix strings.Builder
		prefix.WriteString("⚒️")
		for _, t := range registeredTools {
			if _, used := usedToolNames[t.Definition().Name]; used {
				prefix.WriteString(t.Emoji())
			}
		}
		output = prefix.String() + " " + output
	}

	var allSearchResults []BraveWebResult
	for _, t := range registeredTools {
		if srt, ok := t.(SearchResultsTool); ok {
			allSearchResults = append(allSearchResults, srt.SearchResults()...)
		}
	}

	if apiResponse.ID != "" {
		if saveErr := p.gptService.SetResponseID(c.EffectiveChat, apiResponse.ID); saveErr != nil {
			log.Error().
				Err(saveErr).
				Int64("chat_id", c.EffectiveChat.Id).
				Msg("error saving GPT response ID")
		}
	}

	links := searchLinks(allSearchResults)
	parseMode := ""

	if links != "" {
		// Trim the raw output so the escaped version plus the links section
		// stays within Telegram's limit. Truncating raw (then escaping)
		// guarantees we never cut inside an HTML entity like "&lt;".
		budget := tgUtils.MaxMessageLength - len([]rune(links)) - 1
		runes := []rune(output)
		truncated := false
		for len([]rune(utils.Escape(string(runes)))) > budget {
			overflow := len([]rune(utils.Escape(string(runes)))) - budget
			drop := overflow
			if drop > len(runes) {
				drop = len(runes)
			}
			runes = runes[:len(runes)-drop]
			truncated = true
			if len(runes) == 0 {
				break
			}
		}
		if truncated && len(runes) > 3 {
			runes = append(runes[:len(runes)-3], []rune("...")...)
		}
		output = utils.Escape(string(runes)) + "\n" + links
		parseMode = gotgbot.ParseModeHTML
	} else if len([]rune(output)) > tgUtils.MaxMessageLength {
		output = utils.TruncateText(output, tgUtils.MaxMessageLength-3) + "..."
	}

	_, err = c.EffectiveMessage.ReplyMessage(b, output, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
		ParseMode: parseMode,
	})
	return err
}

func (p *Plugin) reset(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.gptService.ResetResponseID(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error resetting GPT history")
		_, err := c.EffectiveMessage.ReplyMessage(b,
			fmt.Sprintf("❌ Fehler beim Zurücksetzen der GPT-History.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions(),
		)
		return err
	}
	return nil
}

func (p *Plugin) onReset(b *gotgbot.Bot, c plugin.GobotContext) error {
	if err := p.reset(b, c); err != nil {
		return err
	}
	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅",
	})
}

func (p *Plugin) onResetAndRun(b *gotgbot.Bot, c plugin.GobotContext) error {
	if err := p.reset(b, c); err != nil {
		return err
	}
	return p.onGPT(b, c)
}
