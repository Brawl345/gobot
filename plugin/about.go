package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
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
	// TODO: Debug stuff etc. (versioninfo package)
	return c.Reply(plg.key, utils.DefaultSendOptions)
}
