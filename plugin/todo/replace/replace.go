package replace

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "replace"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil
}

func (p *Plugin) Handlers(*telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("^/?s/(.*[^/])/(.*[^/])?/?$"),
			HandlerFunc: onReplace,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile("^/?r/(.*[^/])/(.*[^/])?/?$"),
			HandlerFunc: onRegexReplace,
		},
	}
}

func onReplace(c plugin.GobotContext) error {
	if !c.Message().IsReply() {
		return nil
	}

	text := c.Message().ReplyTo.Text
	if text == "" {
		text = c.Message().ReplyTo.Caption
	}

	if text == "" {
		return nil
	}

	if c.Message().ReplyTo.Sender.ID == c.Bot().Me.ID && strings.HasPrefix(text, "Du meintest wohl:") {
		text = strings.Replace(text, "Du meintest wohl:\n", "", 1)
	}

	var replacement string
	if len(c.Matches) > 1 {
		replacement = c.Matches[2]
	}

	text = strings.ReplaceAll(text, c.Matches[1], replacement)

	_, err := c.Bot().Reply(c.Message().ReplyTo, "<b>Du meintest wohl:</b>\n"+text, utils.DefaultSendOptions)
	return err
}

func onRegexReplace(c plugin.GobotContext) error {
	if !c.Message().IsReply() {
		return nil
	}

	text := c.Message().ReplyTo.Text
	if text == "" {
		text = c.Message().ReplyTo.Caption
	}

	if text == "" {
		return nil
	}

	if c.Message().ReplyTo.Sender.ID == c.Bot().Me.ID && strings.HasPrefix(text, "Du meintest wohl:") {
		text = strings.Replace(text, "Du meintest wohl:\n", "", 1)
	}

	re, err := regexp.Compile(c.Matches[1])
	if err != nil {
		return c.Reply(fmt.Sprintf("❌ Fehler beim Erstellen des regulären Ausdrucks: <code>%v</code>", err),
			utils.DefaultSendOptions)
	}

	text = re.ReplaceAllString(text, c.Matches[2])
	_, err = c.Bot().Reply(c.Message().ReplyTo, "<b>Du meintest wohl:</b>\n"+text, utils.DefaultSendOptions)
	return err
}
