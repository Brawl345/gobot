package manager

import (
	"fmt"
	"github.com/Brawl345/gobot/bot"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/utils"
	"regexp"
)

var log = logger.NewLogger("manager")

type Plugin struct {
	*bot.Plugin
}

func (*Plugin) GetName() string {
	return "manager"
}

func (plg *Plugin) GetCommandHandlers() []bot.CommandHandler {
	return []bot.CommandHandler{
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/enable(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnEnable,
			AdminOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/disable(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnDisable,
			AdminOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/enable_chat(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnEnableInChat,
			AdminOnly: true,
		},
		{
			Command:   regexp.MustCompile(fmt.Sprintf(`^/disable_chat(?:@%s)? (.+)$`, plg.Bot.Me.Username)),
			Handler:   plg.OnDisableInChat,
			AdminOnly: true,
		},
	}
}

func (plg *Plugin) OnEnable(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	err := plg.Bot.EnablePlugin(pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Msg("Failed to enable plugin")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde aktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnEnableInChat(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	err := plg.Bot.EnablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to enable plugin in chat")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat wieder aktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDisable(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	if pluginName == "manager" {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	err := plg.Bot.DisablePlugin(pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Msg("Failed to disable plugin")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde deaktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDisableInChat(c bot.NextbotContext) error {
	pluginName := c.Matches[1]

	if pluginName == "manager" {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	err := plg.Bot.DisablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to disable plugin in chat")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat deaktiviert", utils.DefaultSendOptions)
}
