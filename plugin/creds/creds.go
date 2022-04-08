package creds

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("creds")

type Plugin struct {
	credentialService models.CredentialService
}

func New(credentialService models.CredentialService) *Plugin {
	return &Plugin{
		credentialService: credentialService,
	}
}

func (*Plugin) Name() string {
	return "creds"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnGet,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds_add(?:@%s)? ([^\s]+) (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnAdd,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds_del(?:@%s)? ([^\s]+)$`, botInfo.Username)),
			HandlerFunc: plg.OnDelete,
			AdminOnly:   true,
		},
		&plugin.CallbackHandler{
			HandlerFunc: plg.OnHide,
			Trigger:     regexp.MustCompile(`^creds_hide$`),
			AdminOnly:   true,
		},
	}
}

func (plg *Plugin) OnGet(c plugin.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	creds, err := plg.credentialService.GetAllCredentials()

	if err != nil {
		log.Err(err).Send()
		return c.Reply("❌ Fehler beim Abrufen der Schlüssel", utils.DefaultSendOptions)
	}

	if len(creds) == 0 {
		return c.Reply("<i>Noch keine Schlüssel eingetragen</i>", utils.DefaultSendOptions)
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

func (plg *Plugin) OnAdd(c plugin.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	value := c.Matches[2]

	err := plg.credentialService.SetKey(key, value)
	if err != nil {
		log.Err(err).Str("key", key).Send()
		return c.Reply("❌ Fehler beim Speichern des Schlüssels", utils.DefaultSendOptions)
	}

	return c.Reply("✅ Schlüssel gespeichert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDelete(c plugin.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	err := plg.credentialService.DeleteKey(key)

	if err != nil {
		log.Err(err).Str("key", key).Send()
		return c.Reply(err.Error())
	}

	return c.Reply("✅ Schlüssel gelöscht", utils.DefaultSendOptions)
}

func (plg *Plugin) OnHide(c plugin.NextbotContext) error {
	err := c.Bot().Delete(c.Callback().Message)
	if err != nil {
		log.Err(err).Send()
	}
	return c.Respond()
}
