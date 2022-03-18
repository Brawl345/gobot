package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
	"html"
	"regexp"
	"strconv"
	"strings"
)

type IdPlugin struct {
	*bot.Plugin
}

func (plg *IdPlugin) GetName() string {
	return "id"
}

func (plg *IdPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/(?:(?:whoami)|(?:id))(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler: plg.OnId,
		},
	}
}

func (plg *IdPlugin) GetInlineHandlers() []bot.InlineHandler {
	return []bot.InlineHandler{
		{
			Command:             regexp.MustCompile("^(?:whoami|id)$"),
			Handler:             plg.OnIdInline,
			CanBeUsedByEveryone: true,
		},
	}
}

func (plg *IdPlugin) OnId(c bot.NextbotContext) error {
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

func (plg *IdPlugin) OnIdInline(c bot.NextbotContext) error {
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
		CacheTime:  2,
		IsPersonal: true,
	})
}
