package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"regexp"
)

type AboutPlugin struct {
	*bot.Plugin
	key string
}

func (*AboutPlugin) GetName() string {
	return "about"
}

func (plg *AboutPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/about|start(?:@%s)?$`, plg.Bot.Me.Username)),
			Handler: plg.OnAbout,
		},
	}
}

func (plg *AboutPlugin) Init() {
	plg.key = "Super geheimer Text"
}

func (plg *AboutPlugin) OnAbout(c bot.NextbotContext) error {
	return c.Send(plg.key)
}
