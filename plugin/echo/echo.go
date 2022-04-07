package echo

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "echo"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []bot.Handler {
	return []bot.Handler{
		&bot.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/e(?:cho)?(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onEcho,
		},
	}
}

func onEcho(c bot.NextbotContext) error {
	return c.Reply(c.Matches[1], &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
	})
}
