package creds

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("creds")

type Plugin struct {
	credentialService model.CredentialService
}

func New(credentialService model.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (*Plugin) Name() string {
	return "creds"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil // Because it's a superuser plugin
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: p.OnGet,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds_add(?:@%s)? ([^\s]+) (.+)$`, botInfo.Username)),
			HandlerFunc: p.OnAdd,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/creds_del(?:@%s)? ([^\s]+)$`, botInfo.Username)),
			HandlerFunc: p.OnDelete,
			AdminOnly:   true,
		},
		&plugin.CallbackHandler{
			HandlerFunc: p.OnHide,
			Trigger:     regexp.MustCompile(`^creds_hide$`),
			AdminOnly:   true,
		},
	}
}

func (p *Plugin) OnGet(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	creds, err := p.credentialService.GetAllCredentials()

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Send()
		return c.Reply(fmt.Sprintf("❌ Fehler beim Abrufen der Schlüssel.%s", utils.EmbedGUID(guid)),
			utils.DefaultSendOptions)
	}

	if len(creds) == 0 {
		_, err := c.EffectiveMessage.Reply(b, "<i>Noch keine Schlüssel eingetragen</i>", utils.DefaultSendOptions)
		return err
	}

	var sb strings.Builder

	for _, cred := range creds {
		sb.WriteString(fmt.Sprintf("<b>%s</b>:\n<code>%s</code>\n", cred.Name, cred.Value))
	}

	return c.Reply(sb.String(), &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
		ParseMode:             telebot.ModeHTML,
		ReplyMarkup: &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{
					{
						Text: "Verbergen",
						Data: "creds_hide",
					},
				},
			},
		},
	})

}

func (p *Plugin) OnAdd(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	value := c.Matches[2]

	err := p.credentialService.SetKey(key, value)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Error adding key")
		_, err := c.EffectiveMessage.Reply(b, "❌ Fehler beim Speichern des Schlüssels", utils.DefaultSendOptions)
		return err
	}

	_, err := c.EffectiveMessage.Reply(b, "✅ Schlüssel gespeichert. Der Bot muss neu gestartet werden.", utils.DefaultSendOptions)
	return err
}

func (p *Plugin) OnDelete(b *gotgbot.Bot, c plugin.GobotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	err := p.credentialService.DeleteKey(key)

	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Msg("Error deleting key")
		return c.Reply(err.Error())
	}

	_, err := c.EffectiveMessage.Reply(b, "✅ Schlüssel gelöscht. Der Bot muss neu gestartet werden.", utils.DefaultSendOptions)
	return err
}

func (p *Plugin) OnHide(b *gotgbot.Bot, c plugin.GobotContext) error {
	err := c.Bot().Delete(c.Callback().Message)
	if err != nil {
		log.Err(err).Send()
	}
	return c.Respond()
}
