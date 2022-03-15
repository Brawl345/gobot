package plugin

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/utils"
	"regexp"
)

type ManagerPlugin struct {
	*bot.Plugin
}

func (*ManagerPlugin) GetName() string {
	return "manager"
}

func (plg *ManagerPlugin) GetHandlers() []bot.Handler {
	return []bot.Handler{
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/enable(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler: plg.OnEnable,
		},
		{
			Command: regexp.MustCompile(fmt.Sprintf(`^/disable(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler: plg.OnDisable,
		},
	}
}

func (plg *ManagerPlugin) OnEnable(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	err := plg.Bot.EnablePlugin(pluginName)
	if err != nil {
		return c.Send(err.Error(), utils.DefaultSendOptions)
	}
	return c.Send("✅ Plugin wurde aktiviert", utils.DefaultSendOptions)
}

func (plg *ManagerPlugin) OnDisable(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	if pluginName == "manager" {
		return c.Send("❌ Manager kann nicht deaktiviert werden.")
	}

	err := plg.Bot.DisablePlugin(pluginName)
	if err != nil {
		return c.Send(err.Error(), utils.DefaultSendOptions)
	}
	return c.Send("✅ Plugin wurde deaktiviert", utils.DefaultSendOptions)
}
