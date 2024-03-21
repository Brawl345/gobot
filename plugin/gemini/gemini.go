package gemini

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
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
	Temperature        = 0.7
	TopK               = 1
	TopP               = 1
	MaxOutputTokens    = 700
	MaxInputCharacters = 132000 // Should be roughly 30,000 tokens, max input tokens are 30,720
)

var log = logger.New("gemini")

type (
	Plugin struct {
		apiUrlGemini       string
		apiUrlGeminiVision string
		googleVertexAIKey  string
		geminiService      Service
	}

	Service interface {
		GetHistory(chat *gotgbot.Chat) (model.GeminiData, error)
		ResetHistory(chat *gotgbot.Chat) error
		SetHistory(chat *gotgbot.Chat, history string) error
	}
)

func New(credentialService model.CredentialService, geminiService Service) *Plugin {
	googleVertexAIKey, err := credentialService.GetKey("google_vertex_ai_key")
	if err != nil {
		log.Warn().Msg("google_vertex_ai_key not found")
	}

	apiUrlGeminiPro := ApiUrlGemini
	proxyUrlGeminiPro, err := credentialService.GetKey("google_gemini_proxy")
	if err == nil {
		log.Debug().Msg("Using Gemini API proxy for base model")
		apiUrlGeminiPro = proxyUrlGeminiPro
	}

	apiUrlGeminiVision := ApiUrlGeminiVision
	proxyUrlVision, err := credentialService.GetKey("google_gemini_vision_proxy")
	if err == nil {
		log.Debug().Msg("Using Gemini API proxy for Vision")
		apiUrlGeminiVision = proxyUrlVision
	}

	return &Plugin{
		apiUrlGemini:       apiUrlGeminiPro,
		apiUrlGeminiVision: apiUrlGeminiVision,
		googleVertexAIKey:  googleVertexAIKey,
		geminiService:      geminiService,
	}
}

func (p *Plugin) Name() string {
	return "gemini"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(`(?i)^Bot, ([\s\S]+)$`),
			HandlerFunc: p.onGemini,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/geminireset(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onReset,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/geminireset(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.onResetAndRun,
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) onGemini(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.EffectiveMessage.Photo != nil {
		return p.onGeminiVision(b, c)
	}

	if c.EffectiveMessage.ReplyToMessage != nil && c.EffectiveMessage.ReplyToMessage.Photo != nil {
		return p.onGeminiVision(b, c)
	}

	if c.EffectiveMessage.ExternalReply != nil && c.EffectiveMessage.ExternalReply.Photo != nil {
		return p.onGeminiVision(b, c)
	}

	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionTyping, nil)

	var contents []Content
	geminiData, err := p.geminiService.GetHistory(c.EffectiveChat)
	if err != nil {
		log.Error().
			Err(err).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error getting Gemini data")
	}

	if geminiData.History.Valid && geminiData.ExpiresOn.Valid {
		if time.Now().Before(geminiData.ExpiresOn.Time) {
			var history []Content
			err = json.Unmarshal([]byte(geminiData.History.String), &history)
			if err != nil {
				log.Error().
					Err(err).
					Int64("chat_id", c.EffectiveChat.Id).
					Msg("error unmarshaling Gemini data from DB")
			}
			contents = history
		}
	}

	var inputText strings.Builder

	if tgUtils.IsReply(c.EffectiveMessage) && tgUtils.AnyText(c.EffectiveMessage.ReplyToMessage) != "" {
		inputText.WriteString("-- ZUSÄTZLICHER KONTEXT --\n")
		inputText.WriteString("Dies ist zusätzlicher Kontext. Wiederhole diesen nicht wortwörtlich!\n\n")
		inputText.WriteString(fmt.Sprintf("Nachricht von %s", c.EffectiveMessage.ReplyToMessage.From.FirstName))
		if c.EffectiveMessage.ReplyToMessage.From.LastName != "" {
			inputText.WriteString(fmt.Sprintf(" %s", c.EffectiveMessage.ReplyToMessage.From.LastName))
		}
		inputText.WriteString(":\n")
		inputText.WriteString(tgUtils.AnyText(c.EffectiveMessage.ReplyToMessage))

		if c.EffectiveMessage.Quote != nil && c.EffectiveMessage.Quote.Text != "" {
			inputText.WriteString("\n-- Beziehe dich nur auf folgenden Textteil: --\n")
			inputText.WriteString(c.EffectiveMessage.Quote.Text)
		}

		inputText.WriteString("\n-- ZUSÄTZLICHER KONTEXT ENDE --\n")
	}

	inputText.WriteString(c.Matches[1])

	contents = append(contents, Content{
		Role:  RoleUser,
		Parts: []Part{{Text: inputText.String()}},
	})

	request := Request{
		Contents: contents,
		SafetySettings: []SafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_NONE",
			},
		},
		GenerationConfig: GenerationConfig{
			Temperature:     Temperature,
			TopK:            TopK,
			TopP:            TopP,
			MaxOutputTokens: MaxOutputTokens,
		},
	}

	var response Response

	apiUrl, err := url.Parse(p.apiUrlGemini)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("api_url", p.apiUrlGemini).
			Msg("error while parsing api url")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	q := apiUrl.Query()
	q.Add("key", p.googleVertexAIKey)
	apiUrl.RawQuery = q.Encode()

	err = httpUtils.PostRequest(
		apiUrl.String(),
		nil,
		&request,
		&response,
	)

	if err != nil {
		var httpError *httpUtils.HttpError
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 400 {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("url", p.apiUrlGemini).
					Msg("Failed to send POST request, got HTTP code 400")

				err := p.geminiService.ResetHistory(c.EffectiveChat)
				if err != nil {
					log.Error().
						Err(err).
						Int64("chat_id", c.EffectiveChat.Id).
						Msg("error resetting Gemini data")
				}

				_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten, Konversation wird zurückgesetzt.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
				return err
			}

			if httpError.StatusCode == 429 {
				_, err := c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions())
				return err
			}
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			_, err := c.EffectiveMessage.Reply(b, "❌ Timeout, bitte erneut versuchen.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", p.apiUrlGemini).
			Msg("Failed to send POST request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 ||
		response.Candidates[0].Content.Parts[0].Text == "" {
		log.Error().
			Str("url", p.apiUrlGemini).
			Msg("Got no answer from Gemini")
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Antwort von Gemini erhalten (eventuell gefiltert).", utils.DefaultSendOptions())
		return err
	}

	output := response.Candidates[0].Content.Parts[0].Text

	contents = append(contents, Content{
		Role: RoleModel,
		Parts: []Part{{
			Text: output,
		}},
	})

	inputChars := 0
	for _, content := range contents {
		for _, part := range content.Parts {
			inputChars += len(part.Text)
		}
	}

	if inputChars > MaxInputCharacters {
		err = p.geminiService.ResetHistory(c.EffectiveChat)
		if err != nil {
			log.Error().
				Err(err).
				Int64("chat_id", c.EffectiveChat.Id).
				Msg("error resetting Gemini data")
		}
	} else {
		jsonData, err := json.Marshal(&contents)
		if err != nil {
			log.Error().
				Err(err).
				Int64("chat_id", c.EffectiveChat.Id).
				Msg("error marshalling Gemini data")
		} else {
			err = p.geminiService.SetHistory(c.EffectiveChat, string(jsonData))
			if err != nil {
				log.Error().
					Err(err).
					Int64("chat_id", c.EffectiveChat.Id).
					Msg("error saving Gemini data")
			}
		}
	}

	if len(output) > tgUtils.MaxMessageLength {
		if inputChars > tgUtils.MaxMessageLength {
			output = output[:tgUtils.MaxMessageLength-70] + "..." // More space for the message below
		} else {
			output = output[:tgUtils.MaxMessageLength-9] + "..."
		}
	}

	if inputChars > MaxInputCharacters {
		output += "\n\n(Token-Limit fast erreicht, Konversation wurde zurückgesetzt)"
	}

	_, err = c.EffectiveMessage.Reply(b, output, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
	})

	return err
}

func (p *Plugin) onGeminiVision(b *gotgbot.Bot, c plugin.GobotContext) error {
	// NOTE: Multiturn chat is not enabled for Gemini Pro Vision

	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)

	var bestResolution *gotgbot.PhotoSize

	var inputText strings.Builder

	if c.EffectiveMessage.Photo != nil {
		bestResolution = tgUtils.GetBestResolution(c.EffectiveMessage.Photo)
	} else if c.EffectiveMessage.ReplyToMessage != nil && c.EffectiveMessage.ReplyToMessage.Photo != nil {
		bestResolution = tgUtils.GetBestResolution(c.EffectiveMessage.ReplyToMessage.Photo)
		if c.EffectiveMessage.ReplyToMessage.Caption != "" {
			inputText.WriteString("-- ZUSÄTZLICHER KONTEXT --\n")
			inputText.WriteString("Dies ist zusätzlicher Kontext. Wiederhole diesen nicht wortwörtlich!\n\n")
			inputText.WriteString(fmt.Sprintf("Nachricht von %s", c.EffectiveMessage.ReplyToMessage.From.FirstName))
			if c.EffectiveMessage.ReplyToMessage.From.LastName != "" {
				inputText.WriteString(fmt.Sprintf(" %s", c.EffectiveMessage.ReplyToMessage.From.LastName))
			}
			inputText.WriteString(":\n")
			inputText.WriteString(tgUtils.AnyText(c.EffectiveMessage.ReplyToMessage))

			if c.EffectiveMessage.Quote != nil && c.EffectiveMessage.Quote.Text != "" {
				inputText.WriteString("\n-- Beziehe dich nur auf folgenden Textteil: --\n")
				inputText.WriteString(c.EffectiveMessage.Quote.Text)
			}

			inputText.WriteString("\n-- ZUSÄTZLICHER KONTEXT ENDE --\n")
		}

	} else if c.EffectiveMessage.ExternalReply != nil && c.EffectiveMessage.ExternalReply.Photo != nil {
		bestResolution = tgUtils.GetBestResolution(c.EffectiveMessage.ExternalReply.Photo)
	}
	fileSize := bestResolution.FileSize
	inputText.WriteString(c.Matches[1])

	// This should never happen because the limit for photos sent through Telegram is ~5-10 MB
	if fileSize > tgUtils.MaxFilesizeDownload {
		log.Warn().
			Msgf("File is too big: %d", fileSize)
		_, err := c.EffectiveMessage.Reply(b, "❌Das Bild ist zu groß.", utils.DefaultSendOptions())
		return err
	}

	file, err := httpUtils.DownloadFile(b, bestResolution.FileId)
	if err != nil {
		log.Err(err).
			Interface("photo", bestResolution).
			Msg("Failed to get photo from Telegram")
		_, err := c.EffectiveMessage.Reply(b, "❌ Konnte Bild nicht von Telegram herunterladen.", utils.DefaultSendOptions())
		return err
	}

	defer func(file io.ReadCloser) {
		err := file.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close file")
		}
	}(file)

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Err(err).
			Msg("Failed to read body")
		_, err := c.EffectiveMessage.Reply(b, "❌ Konnte Bild nicht lesen.", utils.DefaultSendOptions())
		return err
	}

	var contents []Content
	contents = append(contents, Content{
		Role: RoleUser,
		Parts: []Part{
			{
				Text: inputText.String(),
			},
			{
				InlineData: &InlineData{
					MimeType: "image/jpeg", // Images sent through Telegram are always JPEGs
					Data:     base64.StdEncoding.EncodeToString(fileData),
				},
			},
		},
	})

	request := Request{
		Contents: contents,
		SafetySettings: []SafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_NONE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_NONE",
			},
		},
		GenerationConfig: GenerationConfig{
			Temperature:     Temperature,
			TopK:            TopK,
			TopP:            TopP,
			MaxOutputTokens: MaxOutputTokens,
		},
	}

	var response Response

	_, _ = c.EffectiveChat.SendAction(b, tgUtils.ChatActionUploadPhoto, nil)

	apiUrl, err := url.Parse(p.apiUrlGeminiVision)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("api_url", p.apiUrlGeminiVision).
			Msg("error while parsing api url")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	q := apiUrl.Query()
	q.Add("key", p.googleVertexAIKey)
	apiUrl.RawQuery = q.Encode()

	err = httpUtils.PostRequest(
		apiUrl.String(),
		nil,
		&request,
		&response,
	)

	if err != nil {
		var httpError *httpUtils.HttpError
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 400 {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("url", p.apiUrlGeminiVision).
					Msg("Failed to send POST request, got HTTP code 400")

				_, err = c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
				return err
			}

			if httpError.StatusCode == 429 {
				_, err := c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions())
				return err
			}
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			_, err := c.EffectiveMessage.Reply(b, "❌ Timeout, bitte erneut versuchen.", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", p.apiUrlGemini).
			Msg("Failed to send POST request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 ||
		response.Candidates[0].Content.Parts[0].Text == "" {
		log.Error().
			Str("url", p.apiUrlGemini).
			Msg("Got no answer from Gemini")
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Antwort von Gemini erhalten (eventuell gefiltert).", utils.DefaultSendOptions())
		return err
	}

	output := response.Candidates[0].Content.Parts[0].Text

	if len(output) > tgUtils.MaxMessageLength {
		output = output[:tgUtils.MaxMessageLength-9] + "..."
	}

	_, err = c.EffectiveMessage.Reply(b, output, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
	})

	return err
}

func (p *Plugin) reset(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.geminiService.ResetHistory(c.EffectiveChat)
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("error resetting history")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Fehler beim Zurücksetzen der Gemini-History.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}
	return nil
}

func (p *Plugin) onReset(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.reset(b, c)
	if err != nil {
		return err
	}

	return tgUtils.AddRectionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅",
	})
}

func (p *Plugin) onResetAndRun(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.reset(b, c)
	if err != nil {
		return err
	}
	return p.onGemini(b, c)
}
