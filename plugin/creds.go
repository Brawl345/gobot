package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
	"log"
	"regexp"
	"strings"
)

type CredsPlugin struct {
	*bot.Plugin
}

func (*CredsPlugin) GetName() string {
	return "creds"
}

func (plg *CredsPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/creds(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler:   plg.OnGet,
			AdminOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/creds_add(?:@%s)? ([^\s]+) (.+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnAdd,
			AdminOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/creds_del(?:@%s)? ([^\s]+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnDelete,
			AdminOnly: true,
		},
	}
}

func (plg *CredsPlugin) GetCallbackHandlers() []bot.CallbackHandler {
	return []bot.CallbackHandler{
		{
			Command:   regexp.MustCompile(`^creds_hide$`),
			Handler:   plg.OnHide,
			AdminOnly: true,
		},
	}
}

func (plg *CredsPlugin) OnGet(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	creds, err := plg.Bot.DB.Credentials.GetAllCredentials()

	if err != nil {
		log.Println(err)
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

func (plg *CredsPlugin) OnAdd(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	value := c.Matches[2]

	err := plg.Bot.DB.Credentials.SetKey(key, value)
	if err != nil {
		log.Println(err)
		return c.Reply("❌ Fehler beim Speichern des Schlüssels", utils.DefaultSendOptions)
	}

	return c.Reply("✅ Schlüssel gespeichert", utils.DefaultSendOptions)
}

func (plg *CredsPlugin) OnDelete(c bot.NextbotContext) error {
	if c.Message().FromGroup() {
		return nil
	}

	key := c.Matches[1]
	err := plg.Bot.DB.Credentials.DeleteKey(key)

	if err != nil {
		log.Println(err)
		return c.Reply(err.Error())
	}

	return c.Reply("✅ Schlüssel gelöscht", utils.DefaultSendOptions)
}

func (plg *CredsPlugin) OnHide(c bot.NextbotContext) error {
	plg.Bot.Delete(c.Callback().Message)
	return c.Respond()
}
