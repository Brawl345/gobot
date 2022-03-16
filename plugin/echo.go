package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
	"regexp"
)

type EchoPlugin struct {
	*bot.Plugin
}

func (*EchoPlugin) GetName() string {
	return "echo"
}

func (plg *EchoPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/e(?:cho)?(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler: plg.OnEcho,
		},
	}
}

func (plg *EchoPlugin) OnEcho(c bot.NextbotContext) error {
	return c.Reply(c.Matches[1], &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
	})
}
