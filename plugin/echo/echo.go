package echo

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/plugin"
	"gopkg.in/telebot.v3"
)

type Plugin struct{}

func New() *Plugin {
	return &Plugin{}
}

func (*Plugin) Name() string {
	return "echo"
}

func (p *Plugin) Commands() []telebot.Command {
	return []telebot.Command{
		{
			Text:        "echo",
			Description: "<Text> - Echo... echo... echo...",
		},
	}
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/e(?:cho)?(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: onEcho,
		},
	}
}

func onEcho(c plugin.GobotContext) error {
	return c.Reply(c.Matches[1], &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
	})
}
