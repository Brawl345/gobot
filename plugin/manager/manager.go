package manager

import (
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("manager")

type (
	Plugin struct {
		managerService Service
	}

	Service interface {
		EnablePlugin(name string) error
		EnablePluginForChat(chat *telebot.Chat, name string) error
		DisablePlugin(name string) error
		DisablePluginForChat(chat *telebot.Chat, name string) error
	}
)

func New(service Service) *Plugin {
	return &Plugin{
		managerService: service,
	}
}

func (*Plugin) Name() string {
	return "manager"
}

func (plg *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/enable(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnEnable,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/disable(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnDisable,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/enable_chat(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnEnableInChat,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`^/disable_chat(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: plg.OnDisableInChat,
			AdminOnly:   true,
		},
	}
}

func (plg *Plugin) OnEnable(c plugin.NextbotContext) error {
	pluginName := c.Matches[1]

	err := plg.managerService.EnablePlugin(pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Msg("Failed to enable plugin")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde aktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnEnableInChat(c plugin.NextbotContext) error {
	pluginName := c.Matches[1]

	err := plg.managerService.EnablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to enable plugin in chat")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat wieder aktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDisable(c plugin.NextbotContext) error {
	pluginName := c.Matches[1]

	if pluginName == "manager" {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	err := plg.managerService.DisablePlugin(pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Msg("Failed to disable plugin")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde deaktiviert", utils.DefaultSendOptions)
}

func (plg *Plugin) OnDisableInChat(c plugin.NextbotContext) error {
	pluginName := c.Matches[1]

	if pluginName == "manager" {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	err := plg.managerService.DisablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		log.Err(err).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to disable plugin in chat")
		return c.Reply(err.Error(), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat deaktiviert", utils.DefaultSendOptions)
}
