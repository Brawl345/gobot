package id

import (
	"fmt"
	"html"
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

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
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
func onId(c plugin.GobotContext) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Du bist <b>%s", html.EscapeString(c.Sender().FirstName)))
	if c.Sender().LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", html.EscapeString(c.Sender().LastName)))
	}
	sb.WriteString("</b> ")
	sb.WriteString(fmt.Sprintf("<code>[%d]</code>", c.Sender().ID))

	if c.Sender().Username != "" {
		sb.WriteString(fmt.Sprintf(" <b>(@%s)</b>", c.Sender().Username))
	}

	if c.Message().FromGroup() {
		sb.WriteString(fmt.Sprintf("\nGruppe: <b>%s</b> <code>[%d]</code>",
			html.EscapeString(c.Chat().Title),
			c.Chat().ID,
		))
	}

	return c.Reply(sb.String(), utils.DefaultSendOptions)
}

func onIdInline(c plugin.GobotContext) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s", html.EscapeString(c.Sender().FirstName)))
	if c.Sender().LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", html.EscapeString(c.Sender().LastName)))
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
