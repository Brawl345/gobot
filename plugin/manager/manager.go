package manager

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
	"gopkg.in/telebot.v3"
)

var log = logger.New("manager")

type (
	Plugin struct {
		managerService models.ManagerService
	}
)

func New(service models.ManagerService) *Plugin {
	return &Plugin{
		managerService: service,
	}
}

func (*Plugin) Name() string {
	return "manager"
}

func (p *Plugin) Commands() []telebot.Command {
	return nil // Because it's a superuser plugin
}

func (p *Plugin) Handlers(botInfo *telebot.User) []plugin.Handler {
	return []plugin.Handler{
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/enable(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.OnEnable,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/disable(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.OnDisable,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/enable_chat(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.OnEnableInChat,
			AdminOnly:   true,
		},
		&plugin.CommandHandler{
			Trigger:     regexp.MustCompile(fmt.Sprintf(`(?i)^/disable_chat(?:@%s)? (.+)$`, botInfo.Username)),
			HandlerFunc: p.OnDisableInChat,
			AdminOnly:   true,
		},
	}
}

func (p *Plugin) OnEnable(c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if p.managerService.IsPluginEnabled(pluginName) {
		return c.Reply("💡 Plugin ist bereits aktiv", utils.DefaultSendOptions)
	}

	err := p.managerService.EnablePlugin(pluginName)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Reply("❌ Plugin existiert nicht", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Msg("Failed to enable plugin")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde aktiviert", utils.DefaultSendOptions)
}

func (p *Plugin) OnEnableInChat(c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if !p.managerService.IsPluginDisabledForChat(c.Chat(), pluginName) {
		return c.Reply("💡 Plugin ist für diesen Chat schon aktiv", utils.DefaultSendOptions)
	}

	err := p.managerService.EnablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Reply("❌ Plugin existiert nicht", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to enable plugin in chat")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat wieder aktiviert", utils.DefaultSendOptions)
}

func (p *Plugin) OnDisable(c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if pluginName == p.Name() {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	if !p.managerService.IsPluginEnabled(pluginName) {
		return c.Reply("💡 Plugin ist nicht aktiv", utils.DefaultSendOptions)
	}

	err := p.managerService.DisablePlugin(pluginName)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Msg("Failed to disable plugin")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde deaktiviert", utils.DefaultSendOptions)
}

func (p *Plugin) OnDisableInChat(c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if pluginName == p.Name() {
		return c.Reply("❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions)
	}

	if p.managerService.IsPluginDisabledForChat(c.Chat(), pluginName) {
		return c.Reply("💡 Plugin ist für diesen Chat schon deaktiviert", utils.DefaultSendOptions)
	}

	err := p.managerService.DisablePluginForChat(c.Chat(), pluginName)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Reply("❌ Plugin existiert nicht", utils.DefaultSendOptions)
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Int64("chat_id", c.Chat().ID).
			Msg("Failed to disable plugin in chat")
		return c.Reply(fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions)
	}
	return c.Reply("✅ Plugin wurde für diesen Chat deaktiviert", utils.DefaultSendOptions)
}
