package summarize

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
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

	"github.com/go-shiori/go-readability"
)

const (
	MinArticleLength        = 500
	DefaultApiUrl           = "https://api.openai.com/v1/chat/completions"
	DefaultApiModel         = "gpt-4o-mini"
	DefaultMaxArticleLength = 128000 * 4.8
	MaxOutputTokens         = 1000
	PresencePenalty         = 1.0
	Temperature             = 0.3
	SystemPrompt            = "Fasse den folgenden Artikel in fünf kurzen Stichpunkten zusammen. Antworte IMMER nur Deutsch. Formatiere deine Ausgabe wie folgt:\n" +
		"Der Artikel handelt von [Zusammenfassung in einem Satz]\n\n" +
		"- [Stichpunkt 1]..."
)

var log = logger.New("summarize")

type Plugin struct {
	credentialService model.CredentialService
}

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (p *Plugin) Name() string {
	return "summarize"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "su",
			Description: "<URL> - Artikel zusammenfassen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/su(?:mmarize)?(?:@%s)? .+$`, botInfo.Username)),
			HandlerFunc: p.onSummarize,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/su(?:mmarize)?(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onReply,
		},
	}
}

func (p *Plugin) onSummarize(b *gotgbot.Bot, c plugin.GobotContext) error {
	return p.summarize(b, c, c.EffectiveMessage)
}

func (p *Plugin) onReply(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !tgUtils.IsReply(c.EffectiveMessage) {
		log.Debug().
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Msg("Message is not a reply")
		return nil
	}

	return p.summarize(b, c, c.EffectiveMessage.ReplyToMessage)
}

func (p *Plugin) summarize(b *gotgbot.Bot, c plugin.GobotContext, msg *gotgbot.Message) error {
	_, _ = c.EffectiveChat.SendAction(b, gotgbot.ChatActionTyping, nil)

	apiKey := p.credentialService.GetKey("summarize_api_key")
	if apiKey == "" {
		log.Warn().Msg("summarize_api_key not found")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>summarize_api_key</code> fehlt.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	chatModel := p.credentialService.GetKey("summarize_model")
	if chatModel == "" {
		chatModel = DefaultApiModel
	}

	// Must be the direct URL to an OpenAI-compatible API - nothing will be appended. Slashes at the end will be removed.
	// Examples:
	// 	OpenAI (default): https://api.openai.com/v1/chat/completions
	// 	Gemini: https://generativelanguage.googleapis.com/v1beta/openai/chat/completions
	// 	Mistral: https://api.mistral.ai/v1/chat/completions
	// 	Groq: https://api.groq.com/openai/v1/chat/completions
	// 	Cloudflare AI Gateway: https://gateway.ai.cloudflare.com/v1/ACCOUNT_TAG/openai/chat/completions
	apiUrl := p.credentialService.GetKey("summarize_api_url")
	if apiUrl == "" {
		apiUrl = DefaultApiUrl
	}

	if !strings.HasPrefix(apiUrl, "http://") && !strings.HasPrefix(apiUrl, "https://") {
		log.Warn().Msg("summarize_api_url is invalid")
		_, err := c.EffectiveMessage.Reply(b,
			"❌ <code>summarize_api_url</code> ist ungültig.",
			utils.DefaultSendOptions(),
		)
		return err
	}

	if strings.HasSuffix(apiUrl, "/") {
		apiUrl = apiUrl[:len(apiUrl)-1]
	}

	var ctxWindow float64
	ctxWindowStr := p.credentialService.GetKey("summarize_ctx_window")
	if ctxWindowStr != "" {
		var err error
		ctxWindow, err = strconv.ParseFloat(ctxWindowStr, 64)
		if err != nil {
			log.Err(err).Msg("Failed to parse summarize_ctx_window")
			_, err := c.EffectiveMessage.Reply(b,
				"❌ <code>summarize_ctx_window</code> ist ungültig.",
				utils.DefaultSendOptions(),
			)
			return err
		}
	}

	maxArticleLength := ctxWindow * 4.8 // Roughly with overhead
	if maxArticleLength == 0 {
		maxArticleLength = DefaultMaxArticleLength
	}

	if maxArticleLength < MinArticleLength {
		maxArticleLength = MinArticleLength
	}

	var urls []string
	for _, entity := range tgUtils.ParseAnyEntityTypes(msg, []tgUtils.EntityType{tgUtils.EntityTypeURL, tgUtils.EntityTextLink}) {
		urls = append(urls, entity.Url)
	}

	if len(urls) == 0 {
		_, err := msg.Reply(b, "❌ Keine Links gefunden", utils.DefaultSendOptions())
		return err
	}

	url := urls[0] // only summarize the first URL for now

	article, err := readability.FromURL(url, 10*time.Second)
	if err != nil {
		log.Err(err).
			Str("url", url).
			Msg("Failed to extract text content from URL")

		_, err := msg.Reply(b,
			fmt.Sprintf("❌ Text konnte nicht extrahiert werden: <code>%v</code>", utils.Escape(err.Error())),
			utils.DefaultSendOptions())
		return err
	}

	if len(article.TextContent) < MinArticleLength {
		_, err := msg.Reply(b,
			"❌ Artikel-Inhalt ist zu kurz.",
			utils.DefaultSendOptions())
		return err
	}

	if len(article.TextContent) > int(math.Ceil(maxArticleLength)) {
		_, err := msg.Reply(b,
			"❌ Artikel-Inhalt ist zu lang.",
			utils.DefaultSendOptions())
		return err
	}

	request := Request{
		Model: chatModel,
		Messages: []ApiMessage{
			{
				Role:    System,
				Content: SystemPrompt,
			},
			{
				Role:    User,
				Content: article.TextContent,
			},
		},
		PresencePenalty: PresencePenalty,
		MaxTokens:       MaxOutputTokens,
		Temperature:     Temperature,
	}

	var response Response
	var httpError *httpUtils.HttpError

	err = httpUtils.MakeRequest(httpUtils.RequestOptions{
		Method: httpUtils.MethodPost,
		URL:    apiUrl,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		},
		Body:     &request,
		Response: &response,
	})

	if err != nil {
		if errors.As(err, &httpError) {
			if httpError.StatusCode == http.StatusTooManyRequests {
				_, err := c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions())
				return err
			}
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("Failed to send POST request")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if response.Error.Type != "" {
		guid := xid.New().String()
		log.Error().
			Str("guid", guid).
			Str("url", url).
			Str("message", response.Error.Message).
			Str("type", response.Error.Type).
			Msg("Got error from model API")
		_, err := c.EffectiveMessage.Reply(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions())
		return err
	}

	if len(response.Choices) == 0 || (len(response.Choices) > 0 && response.Choices[0].Message.Content == "") {
		log.Error().
			Str("url", url).
			Msg("Got no answer from ChatGPT")
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Antwort vom KI-Modell erhalten", utils.DefaultSendOptions())
		return err
	}

	var sb strings.Builder
	sb.WriteString("<b>Zusammenfassung:</b>\n")
	sb.WriteString(utils.Escape(response.Choices[0].Message.Content))

	_, err = msg.Reply(b, sb.String(), utils.DefaultSendOptions())
	return err
}
