package id

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

const InlineQueryCacheTime = 7200

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "id"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return []gotgbot.BotCommand{
		{
			Command:     "whoami",
			Description: "Deine Telegram-Informationen anzeigen",
		},
	}
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/(?:(?:whoami)|(?:id))(?:@%s)?$`, botInfo.Username)),
			HandlerFunc: onId,
		},
		&plugin.InlineHandler{
			HandlerFunc:         onIdInline,
			Trigger:             regexp.MustCompile("(?i)^(?:whoami|id)$"),
			CanBeUsedByEveryone: true,
		},
	}
}
func onId(b *gotgbot.Bot, c plugin.GobotContext) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Du bist <b>%s", utils.Escape(c.Sender().FirstName)))
	if c.Sender().LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", utils.Escape(c.Sender().LastName)))
	}
	sb.WriteString("</b> ")
	sb.WriteString(fmt.Sprintf("<code>[%d]</code>", c.Sender().ID))

	if c.Sender().Username != "" {
		sb.WriteString(fmt.Sprintf(" <b>(@%s)</b>", c.Sender().Username))
	}

	if c.Message().FromGroup() {
		sb.WriteString(fmt.Sprintf("\nGruppe: <b>%s</b> <code>[%d]</code>",
			utils.Escape(c.Chat().Title),
			c.Chat().ID,
		))
	}

	_, err := c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}

func onIdInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s", utils.Escape(c.Sender().FirstName)))
	if c.Sender().LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", utils.Escape(c.Sender().LastName)))
	}
	sb.WriteString("</b> ")
	sb.WriteString(fmt.Sprintf("<code>[%d]</code>", c.Sender().ID))

	if c.Sender().Username != "" {
		sb.WriteString(fmt.Sprintf(" <b>(@%s)</b>", c.Sender().Username))
	}

	result := &telebot.ArticleResult{
		Title: strconv.FormatInt(c.Sender().ID, 10),
		Text:  sb.String(),
	}
	result.SetContent(&telebot.InputTextMessageContent{
		Text:           sb.String(),
		ParseMode:      telebot.ModeHTML,
		DisablePreview: true,
	})

	return c.Answer(&telebot.QueryResponse{
		Results:    telebot.Results{result},
		CacheTime:  InlineQueryCacheTime,
		IsPersonal: true,
	})
}
