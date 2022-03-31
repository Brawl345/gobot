package echo

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"gopkg.in/telebot.v3"
	"regexp"
)

type Plugin struct {
	*bot.Plugin
}

func (*Plugin) GetName() string {
	return "echo"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/e(?:cho)?(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler: plg.OnEcho,
		},
	}
}

func (plg *Plugin) OnEcho(c bot.NextbotContext) error {
	return c.Reply(c.Matches[1], &telebot.SendOptions{
		AllowWithoutReply:     true,
		DisableWebPagePreview: true,
	})
}
