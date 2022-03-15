package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
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
	return c.Send(c.Matches[1])
}
