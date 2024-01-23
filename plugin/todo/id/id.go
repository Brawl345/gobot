package id

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
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

	sb.WriteString(fmt.Sprintf("Du bist <b>%s", utils.Escape(c.EffectiveUser.FirstName)))
	if c.EffectiveUser.LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", utils.Escape(c.EffectiveUser.LastName)))
	}
	sb.WriteString("</b> ")
	sb.WriteString(fmt.Sprintf("<code>[%d]</code>", c.EffectiveUser.Id))

	if c.EffectiveUser.Username != "" {
		sb.WriteString(fmt.Sprintf(" <b>(@%s)</b>", c.EffectiveUser.Username))
	}

	if c.Message().FromGroup() {
		sb.WriteString(fmt.Sprintf("\nGruppe: <b>%s</b> <code>[%d]</code>",
			utils.Escape(c.EffectiveChat.Title),
			c.EffectiveChat.Id,
		))
	}

	_, err := c.EffectiveMessage.Reply(b, sb.String(), utils.DefaultSendOptions)
	return err
}

func onIdInline(b *gotgbot.Bot, c plugin.GobotContext) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s", utils.Escape(c.EffectiveUser.FirstName)))
	if c.EffectiveUser.LastName != "" {
		sb.WriteString(fmt.Sprintf(" %s", utils.Escape(c.EffectiveUser.LastName)))
	}
	sb.WriteString("</b> ")
	sb.WriteString(fmt.Sprintf("<code>[%d]</code>", c.EffectiveUser.Id))

	if c.EffectiveUser.Username != "" {
		sb.WriteString(fmt.Sprintf(" <b>(@%s)</b>", c.EffectiveUser.Username))
	}

	result := &telebot.ArticleResult{
		Title: strconv.FormatInt(c.EffectiveUser.Id, 10),
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
