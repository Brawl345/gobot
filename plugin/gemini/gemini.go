package gemini

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
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
	Temperature              = 0.8
	TopK                     = 1
	TopP                     = 1
	MaxOutputTokens          = 700
	MaxInputCharacters       = 250000 // Should be roughly 1 mio tokens, max input tokens are 1048576
	TokensPerImage           = 258    // https://ai.google.dev/gemini-api/docs/prompting_with_media?lang=python#video_formats
	DefaultSystemInstruction = "Antworte nur auf Deutsch. Nutze nur Standard-Text, da Markdown für den Nutzer nicht angezeigt wird. Verwende keine Emoji. Bilder-Analyse ist eingeschaltet."
)

var log = logger.New("gemini")

type (
	Plugin struct {
		// Get the key from https://aistudio.google.com/app/apikey
		// NOTE: In Europe you have to set up billing for your project (or route requests through a proxy in the US)
		credentialService model.CredentialService
		geminiService     Service
	}

	Service interface {
		GetHistory(chat *gotgbot.Chat) (model.GeminiData, error)
		ResetHistory(chat *gotgbot.Chat) error
		SetHistory(chat *gotgbot.Chat, history string) error
	}
)

func New(credentialService model.CredentialService, geminiService Service) *Plugin {
	return &Plugin{
		credentialService: credentialService,
		geminiService:     geminiService,
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
			Trigger:     regexp.MustCompile(`(?i)^Bot,? ([\s\S]+)$`),
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
	apiKey := p.credentialService.GetKey("google_generative_language_api_key")
	if apiKey == "" {
		log.Warn().Msg("google_generative_language_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>google_generative_language_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	apiUrlGemini := ApiUrlGemini
	proxyUrlGemini := p.credentialService.GetKey("google_gemini_proxy")
	if proxyUrlGemini != "" {
		log.Debug().Msg("Using Gemini API proxy for base model")
		apiUrlGemini = proxyUrlGemini
	}

	systemInstruction := cmp.Or(p.credentialService.GetKey("google_gemini_system_instruction"), DefaultSystemInstruction)

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

	var photo *gotgbot.PhotoSize
	var inputText strings.Builder

	if tgUtils.IsReply(c.EffectiveMessage) {
		photo = tgUtils.GetBestResolution(c.EffectiveMessage.ReplyToMessage.Photo)
		if c.EffectiveMessage.ReplyToMessage.GetText() != "" {
			inputText.WriteString("-- ZUSÄTZLICHER KONTEXT --\n")
			inputText.WriteString("Dies ist zusätzlicher Kontext. Wiederhole diesen nicht wortwörtlich!\n\n")
			inputText.WriteString(fmt.Sprintf("Nachricht von %s", c.EffectiveMessage.ReplyToMessage.From.FirstName))
			if c.EffectiveMessage.ReplyToMessage.From.LastName != "" {
				inputText.WriteString(fmt.Sprintf(" %s", c.EffectiveMessage.ReplyToMessage.From.LastName))
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

	// If both the reply and the message have a photo, just take the one from the message
	if c.EffectiveMessage.Photo != nil {
		photo = tgUtils.GetBestResolution(c.EffectiveMessage.Photo)
	}

	inputText.WriteString(c.Matches[1])

	parts := []Part{{Text: inputText.String()}}

	//Upload photo first: https://ai.google.dev/gemini-api/docs/prompting_with_media
	if photo != nil {
		_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionUploadPhoto, nil)

		fileSize := photo.FileSize

		// This should never happen because the limit for photos sent through Telegram is ~5-10 MB
		if fileSize > tgUtils.MaxFilesizeDownload {
			log.Warn().
				Msgf("File is too big: %d", fileSize)
			_, err := c.EffectiveMessage.Reply(b, "❌Das Bild ist zu groß.", utils.DefaultSendOptions())
			return err
		}

		file, err := httpUtils.DownloadFile(b, photo.FileId)
		if err != nil {
			log.Err(err).
				Interface("photo", photo).
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

		var fileUploadResponse FileUploadResponse

		err = httpUtils.MakeRequest(httpUtils.RequestOptions{
			Method:   httpUtils.MethodPost,
			URL:      fmt.Sprintf(ApiUrlFileUpload, apiKey),
			Headers:  map[string]string{"Content-Type": "image/jpeg"},
			Body:     file,
			Response: &fileUploadResponse,
		})

		if err != nil {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Str("api_url", ApiUrlFileUpload).
				Msg("error while uploading file")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		if fileUploadResponse.File.MimeType == "" || fileUploadResponse.File.Uri == "" {
			guid := xid.New().String()
			log.Err(err).
				Str("guid", guid).
				Interface("fileUploadResponse", fileUploadResponse).
				Msg("error while uploading file")
			_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
			return err
		}

		parts = append(parts, Part{FileData: &FileData{
			MimeType: fileUploadResponse.File.MimeType,
			FileUri:  fileUploadResponse.File.Uri,
		}})

	}

	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	contents = append(contents, Content{
		Role:  RoleUser,
		Parts: parts,
	})

	request := GenerateContentRequest{
		Contents:          contents,
		SystemInstruction: SystemInstruction{Parts: []Part{{Text: systemInstruction}}},
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

	var response GenerateContentResponse

	apiUrl, err := url.Parse(apiUrlGemini)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("api_url", apiUrlGemini).
			Msg("error while parsing api url")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	q := apiUrl.Query()
	q.Add("key", apiKey)
	apiUrl.RawQuery = q.Encode()

	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method:   httpUtils.MethodPost,
		URL:      apiUrl.String(),
		Body:     &request,
		Response: &response,
	})

	if err != nil {
		var httpError *httpUtils.HttpError
		if errors.As(err, &httpError) {
			if httpError.StatusCode == http.StatusBadRequest {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("url", apiUrlGemini).
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

			if httpError.StatusCode == http.StatusTooManyRequests {
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
			Str("url", apiUrlGemini).
			Msg("Failed to send POST request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 ||
		response.Candidates[0].Content.Parts[0].Text == "" {
		log.Error().
			Str("url", apiUrlGemini).
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
			if part.FileData != nil && part.FileData.FileUri != "" {
				inputChars += TokensPerImage
			}
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
