package replace

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "replace"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil
}

func (p *Plugin) Handlers(*gotgbot.User) []plugin.Handler {
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

func onReplace(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !utils.IsReply(c.EffectiveMessage) {
		return nil
	}

	text := c.EffectiveMessage.ReplyToMessage.Text
	if text == "" {
		text = c.EffectiveMessage.ReplyToMessage.Caption
	}

	if text == "" {
		return nil
	}

	if c.EffectiveMessage.ReplyToMessage.From.Id == b.Id && strings.HasPrefix(text, "Du meintest wohl:") {
		text = strings.Replace(text, "Du meintest wohl:\n", "", 1)
	}

	var replacement string
	if len(c.Matches) > 1 {
		replacement = c.Matches[2]
	}

	text = strings.ReplaceAll(text, c.Matches[1], replacement)

	_, err := c.EffectiveMessage.ReplyToMessage.Reply(b, "<b>Du meintest wohl:</b>\n"+text, &gotgbot.SendMessageOpts{
		ReplyParameters: &gotgbot.ReplyParameters{
			AllowSendingWithoutReply: true,
		},
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
		DisableNotification: true,
		ParseMode:           gotgbot.ParseModeHTML,
	})
	return err
}

func onRegexReplace(b *gotgbot.Bot, c plugin.GobotContext) error {
	if !utils.IsReply(c.EffectiveMessage) {
		return nil
	}

	text := c.EffectiveMessage.ReplyToMessage.Text
	if text == "" {
		text = c.EffectiveMessage.ReplyToMessage.Caption
	}

	if text == "" {
		return nil
	}

	if c.EffectiveMessage.ReplyToMessage.From.Id == b.Id && strings.HasPrefix(text, "Du meintest wohl:") {
		text = strings.Replace(text, "Du meintest wohl:\n", "", 1)
	}

	re, err := regexp.Compile(c.Matches[1])
	if err != nil {
		_, err = c.EffectiveMessage.Reply(b,
			fmt.Sprintf("❌ Fehler beim Erstellen des regulären Ausdrucks: <code>%v</code>", err),
			utils.DefaultSendOptions(),
		)
		return err
	}

	text = re.ReplaceAllString(text, c.Matches[2])
	_, err = c.EffectiveMessage.ReplyToMessage.Reply(b, "<b>Du meintest wohl:</b>\n"+text, utils.DefaultSendOptions())
	return err
}
