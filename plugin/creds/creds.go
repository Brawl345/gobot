package creds

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("creds")

type Plugin struct {
	bot *bot.Nextbot
}

func New(bot *bot.Nextbot) *Plugin {
	return &Plugin{
		bot: bot,
	}
}

func (*Plugin) Name() string {
	return "creds"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []bot.Handler {
	return []bot.Handler{
		&bot.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: plg.OnGet,
			AdminOnly:   true,
		},
		&bot.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds_add(?:@%s)? ([^\s]+) (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnAdd,
			AdminOnly:   true,
		},
		&bot.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/creds_del(?:@%s)? ([^\s]+)$`, botInfo.Username)),
			HandlerFunc: plg.OnDelete,
			AdminOnly:   true,
		},
		&bot.CallbackHandler{
			HandlerFunc: plg.OnHide,
			Trigger:     regexp.MustCompile(`^creds_hide$`),
			AdminOnly:   true,
		},
	}
}

func (plg *Plugin) OnGet(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	creds, err := plg.bot.DB.Credentials.GetAllCredentials()

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

func (plg *Plugin) OnAdd(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	value := c.Matches[2]

	err := plg.bot.DB.Credentials.SetKey(key, value)
	if err != nil {
		log.Err(err).Str("key", key).Send()
		return c.Reply("❌ Fehler beim Speichern des Schlüssels", utils.DefaultSendOptions)
	}

	return c.Reply("✅ Schlüssel gespeichert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDelete(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	err := plg.bot.DB.Credentials.DeleteKey(key)

	if err != nil {
		log.Err(err).Str("key", key).Send()
		return c.Reply(err.Error())
	}

	return c.Reply("✅ Schlüssel gelöscht", utils.DefaultSendOptions)
}

func (plg *Plugin) OnHide(c bot.NextbotContext) error {
	err := plg.bot.Delete(c.Callback().Message)
	if err != nil {
		log.Err(err).Send()
	}
	return c.Respond()
}
