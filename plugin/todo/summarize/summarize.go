package summarize

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"

	"github.com/go-shiori/go-readability"
)

const (
	MinArticleLength = 500
	MaxArticleLength = 60000 // ~12,000 tokens
	MaxTokens        = 1000
	PresencePenalty  = 1.0
	Temperature      = 0.3
	SystemPrompt     = "Summarize the following article in five short bullet points. Always speak in German. If the website needs JavaScript and has a cookie consent banner, reply with a clown emoji."
)

var log = logger.New("summarize")

type Plugin struct {
	apiUrl                   string
	openAIApiKey             string
	cloudflareApiGatewayPath string
}

func New(credentialService model.CredentialService) *Plugin {
	openAIApiKey, err := credentialService.GetKey("openai_api_key")
	if err != nil {
		log.Warn().Msg("openai_api_key not found")
	}

	apiUrl := OpenAIApiUrl

	// In the format "accountTag/gateway"
	cloudflareApiGatewayPath, _ := credentialService.GetKey("openai_cloudflare_ai_gateway")
	if cloudflareApiGatewayPath != "" {
		if strings.HasPrefix(cloudflareApiGatewayPath, "/") {
			cloudflareApiGatewayPath = cloudflareApiGatewayPath[1:]
		}
		if strings.HasSuffix(cloudflareApiGatewayPath, "/") {
			cloudflareApiGatewayPath = cloudflareApiGatewayPath[:len(cloudflareApiGatewayPath)-1]
		}

		apiUrl = fmt.Sprintf(CloudflareGatewayUrl, cloudflareApiGatewayPath)
		log.Debug().Msg("Using Cloudflare AI Gateway")
	}

	return &Plugin{
		apiUrl:                   apiUrl,
		openAIApiKey:             openAIApiKey,
		cloudflareApiGatewayPath: cloudflareApiGatewayPath,
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
			GroupOnly:   true,
		},
	}
}

func (p *Plugin) onSummarize(b *gotgbot.Bot, c plugin.GobotContext) error {
	return p.summarize(c, c.Message())
}

func (p *Plugin) onReply(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !c.Message().IsReply() {
		log.Debug().
			Int64("chat_id", c.EffectiveChat.Id).
			Int64("user_id", c.EffectiveUser.Id).
			Msg("Message is not a reply")
		return nil
	}

	if strings.HasPrefix(c.Message().ReplyTo.Text, "/su") ||
		strings.HasPrefix(c.Message().ReplyTo.Caption, "/su") {
		_, err := c.EffectiveMessage.Reply(b, "😠", utils.DefaultSendOptions)
		return err
	}

	if c.Message().ReplyTo.Sender.IsBot {
		_, err := c.EffectiveMessage.Reply(b, "😠", utils.DefaultSendOptions)
		return err
	}

	return p.summarize(c, c.Message().ReplyTo)
}

func (p *Plugin) summarize(c plugin.GobotContext, msg *telebot.Message) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

	var urls []string
	for _, entity := range utils.AnyEntities(msg) {
		if entity.Type == telebot.EntityURL {
			urls = append(urls, msg.EntityText(entity))
		} else if entity.Type == telebot.EntityTextLink {
			urls = append(urls, entity.URL)
		}
	}

	if len(urls) == 0 {
		_, err := c.Bot().Reply(msg, "❌ Keine Links gefunden", utils.DefaultSendOptions)
		return err
	}

	url := urls[0] // only summarize the first URL for now

	article, err := readability.FromURL(url, 10*time.Second)
	if err != nil {
		log.Err(err).
			Str("url", url).
			Msg("Failed to extract text content from URL")

		_, err := c.Bot().Reply(msg,
			fmt.Sprintf("❌ Text konnte nicht extrahiert werden: <code>%v</code>", utils.Escape(err.Error())),
			utils.DefaultSendOptions)
		return err
	}

	if len(article.TextContent) < MinArticleLength {
		_, err := c.Bot().Reply(msg,
			"❌ Artikel-Inhalt ist zu kurz.",
			utils.DefaultSendOptions)
		return err
	}

	if len(article.TextContent) > MaxArticleLength {
		_, err := c.Bot().Reply(msg,
			"❌ Artikel-Inhalt ist zu lang.",
			utils.DefaultSendOptions)
		return err
	}

	request := Request{
		Model: Model,
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
		MaxTokens:       MaxTokens,
		Temperature:     Temperature,
	}

	var response Response
	var httpError *httpUtils.HttpError
	err = httpUtils.PostRequest(p.apiUrl,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", p.openAIApiKey),
		},
		&request,
		&response)
	if err != nil {

		if errors.As(err, &httpError) {
			if httpError.StatusCode == 429 {
				_, err := c.EffectiveMessage.Reply(b, "❌ Rate-Limit erreicht.", utils.DefaultSendOptions)
				return err
			}
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("url", url).
			Msg("Failed to send POST request")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if response.Error.Type != "" {
		guid := xid.New().String()
		log.Error().
			Str("guid", guid).
			Str("url", url).
			Str("message", response.Error.Message).
			Str("type", response.Error.Type).
			Msg("Got error from OpenAI API")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(response.Choices) == 0 || (len(response.Choices) > 0 && response.Choices[0].Message.Content == "") {
		log.Error().
			Str("url", url).
			Msg("Got no answer from ChatGPT")
		_, err := c.EffectiveMessage.Reply(b, "❌ Keine Antwort von ChatGPT erhalten", utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder
	sb.WriteString("<b>Zusammenfassung:</b>\n")
	sb.WriteString(utils.Escape(response.Choices[0].Message.Content))

	_, err = c.Bot().Reply(msg, sb.String(), utils.DefaultSendOptions)
	return err
}
