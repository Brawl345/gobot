package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
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
		apiUrl            string
		googleVertexAIKey string
		geminiService     Service
	}

	Service interface {
		GetHistory(chat *telebot.Chat) (model.GeminiData, error)
		ResetHistory(chat *telebot.Chat) error
		SetHistory(chat *telebot.Chat, history string) error
	}
)

func New(credentialService model.CredentialService, geminiService Service) *Plugin {
	googleVertexAIKey, err := credentialService.GetKey("google_vertex_ai_key")
	if err != nil {
		log.Warn().Msg("google_vertex_ai_key not found")
	}

	apiUrl := ApiUrl
	proxyUrl, err := credentialService.GetKey("google_gemini_proxy")
	if err == nil {
		log.Debug().Msg("Using Gemini API proxy")
		apiUrl = proxyUrl
	}

	return &Plugin{
		apiUrl:            apiUrl,
		googleVertexAIKey: googleVertexAIKey,
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
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

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

	contents = append(contents, Content{
		Role:  RoleUser,
		Parts: []Part{{Text: c.Matches[1]}},
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
	var httpError *httpUtils.HttpError
	err = httpUtils.PostRequest(
		p.apiUrl+"?key="+p.googleVertexAIKey,
		nil,
		&request,
		&response,
	)

	if err != nil {
		if errors.As(err, &httpError) {
			if httpError.StatusCode == 400 {
				guid := xid.New().String()
				log.Err(err).
					Str("guid", guid).
					Str("url", p.apiUrl).
					Msg("Failed to send POST request, got HTTP code 400")

				err := p.geminiService.ResetHistory(c.EffectiveChat)
				if err != nil {
					log.Error().
						Err(err).
						Int64("chat_id", c.EffectiveChat.Id).
						Msg("error resetting Gemini data")
				}

				return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten, Konversation wird zurückgesetzt.%s", utils.EmbedGUID(guid)),
					utils.DefaultSendOptions)
			}

			if httpError.StatusCode == 429 {
				_, err := c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions)
				return err
			}
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", p.apiUrl).
			Msg("Failed to send POST request")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 ||
		response.Candidates[0].Content.Parts[0].Text == "" {
		log.Error().
			Str("url", p.apiUrl).
			Msg("Got no answer from Gemini")
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Antwort von Gemini erhalten (eventuell gefiltert).", utils.DefaultSendOptions)
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

	if len(output) > utils.MaxMessageLength {
		if inputChars > utils.MaxMessageLength {
			output = output[:utils.MaxMessageLength-70] + "..." // More space for the message below
		} else {
			output = output[:utils.MaxMessageLength-9] + "..."
		}
	}

	if inputChars > MaxInputCharacters {
		output += "\n\n(Token-Limit fast erreicht, Konversation wurde zurückgesetzt)"
	}

	_, err = c.Bot().Reply(c.EffectiveMessage, output, &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
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
		return c.Reply(fmt.Sprintf("❌ Fehler beim Zurücksetzen der Gemini-History.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}
	return nil
}

func (p *Plugin) onReset(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.reset(c)
	if err != nil {
		return err
	}
	_, err := c.EffectiveMessage.Reply(b, "✅", utils.DefaultSendOptions)
	return err
}

func (p *Plugin) onResetAndRun(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.reset(c)
	if err != nil {
		return err
	}
	return p.onGemini(c)
}
