package cleverbot

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/Brawl345/gobot/utils/httpUtils"
	"github.com/rs/xid"
)

const BaseUrl = "https://www.cleverbot.com/getreply"

var log = logger.New("cleverbot")

type (
	Plugin struct {
		apiKey           string
		cleverbotService Service
	}

	Service interface {
		SetState(chat *telebot.Chat, state string) error
		ResetState(chat *telebot.Chat) error
		GetState(chat *telebot.Chat) (string, error)
	}
)

func New(credentialService model.CredentialService, cleverbotService Service) *Plugin {
	apiKey, err := credentialService.GetKey("cleverbot_api_key")
	if err != nil {
		log.Warn().Msg("cleverbot_api_key not found")
	}
	return &Plugin{
		apiKey:           apiKey,
		cleverbotService: cleverbotService,
	}
}

func (p *Plugin) Name() string {
	return "cleverbot"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "cbot",
			Description: "<Text> - Befrag den Cleverbot",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/cbot(?:@%s)? ([\s\S]+)$`, botInfo.Username)),
			HandlerFunc: p.onCleverbot,
			GroupOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/cbotreset(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.onReset,
			GroupOnly:   true,
			AdminOnly:   true,
		},
	}
}

func (p *Plugin) onCleverbot(b *gotgbot.Bot, c plugin.GobotContext) error {
	_, _ = c.EffectiveChat.SendAction(b, utils.ChatActionTyping, nil)

	state, err := p.cleverbotService.GetState(c.Chat())
	if err != nil {
		log.Error().
			Err(err).
			Int64("chat_id", c.Chat().ID).
			Msg("error getting state")
	}

	requestUrl := fmt.Sprintf(
		"%s?key=%s&input=%s&cs=%s",
		BaseUrl,
		p.apiKey,
		url.QueryEscape(c.Matches[1]),
		url.QueryEscape(state),
	)

	var response Response
	err = httpUtils.GetRequest(
		requestUrl,
		&response,
	)

	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Str("url", requestUrl).
			Msg("error contacting cleverbot")
		return c.Reply(fmt.Sprintf("❌ Fehler bei der Kommunikation mit dem Cleverbot.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if response.Output == "" {
		err := p.cleverbotService.ResetState(c.Chat())
		if err != nil {
			log.Error().
				Err(err).
				Int64("chat_id", c.Chat().ID).
				Msg("error resetting state")
		}
		return c.Reply("😴 Cleverbot müde...", &telebot.SendOptions{
			AllowWithoutReply: true,
		})
	}

	if len(response.State) > 16777200 { // Enough...
		err = p.cleverbotService.ResetState(c.Chat())
		if err != nil {
			log.Error().
				Err(err).
				Int64("chat_id", c.Chat().ID).
				Msg("error resetting state")
		}
	} else {
		err = p.cleverbotService.SetState(c.Chat(), response.State)
		if err != nil {
			log.Error().
				Err(err).
				Int64("chat_id", c.Chat().ID).
				Str("cs", response.State).
				Msg("error setting state")
		}
	}

	return c.Reply(response.Output, &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
	})
}

func (p *Plugin) onReset(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := p.cleverbotService.ResetState(c.Chat())
	if err != nil {
		guid := xid.New().String()
		log.Error().
			Err(err).
			Str("guid", guid).
			Int64("chat_id", c.Chat().ID).
			Msg("error resetting state")
		return c.Reply(fmt.Sprintf("❌ Fehler beim Zurücksetzen des Cleverbot-Status.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	_, err := c.EffectiveMessage.Reply(b, "✅", utils.DefaultSendOptions)
	return err
}
