package manager

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils"
	"github.com/rs/xid"
)

var log = logger.New("manager")

type (
	Plugin struct {
		managerService model.ManagerService
	}
)

func New(service model.ManagerService) *Plugin {
	return &Plugin{
		managerService: service,
	}
}

func (*Plugin) Name() string {
	return "manager"
}

func (p *Plugin) Commands() []gotgbot.BotCommand {
	return nil // Because it's a superuser plugin
}

func (p *Plugin) Handlers(botInfo *gotgbot.User) []plugin.Handler {
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

func (p *Plugin) OnEnable(b *gotgbot.Bot, c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if p.managerService.IsPluginEnabled(pluginName) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Plugin ist bereits aktiv", utils.DefaultSendOptions())
		return err
	}

	err := p.managerService.EnablePlugin(pluginName)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err = c.EffectiveMessage.ReplyMessage(b, "❌ Plugin existiert nicht", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Msg("Failed to enable plugin")
		_, err = c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Plugin wurde aktiviert",
	})
}

func (p *Plugin) OnEnableInChat(b *gotgbot.Bot, c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if !p.managerService.IsPluginDisabledForChat(c.EffectiveChat, pluginName) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Plugin ist für diesen Chat schon aktiv", utils.DefaultSendOptions())
		return err
	}

	err := p.managerService.EnablePluginForChat(c.EffectiveChat, pluginName)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err = c.EffectiveMessage.ReplyMessage(b, "❌ Plugin existiert nicht", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to enable plugin in chat")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Plugin wurde für diesen Chat wieder aktiviert",
	})
}

func (p *Plugin) OnDisable(b *gotgbot.Bot, c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if pluginName == p.Name() {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions())
		return err
	}

	if !p.managerService.IsPluginEnabled(pluginName) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Plugin ist nicht aktiv", utils.DefaultSendOptions())
		return err
	}

	err := p.managerService.DisablePlugin(pluginName)
	if err != nil {
		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Msg("Failed to disable plugin")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Plugin wurde deaktiviert",
	})
}

func (p *Plugin) OnDisableInChat(b *gotgbot.Bot, c plugin.GobotContext) error {
	pluginName := c.Matches[1]

	if pluginName == p.Name() {
		_, err := c.EffectiveMessage.ReplyMessage(b, "❌ Manager kann nicht deaktiviert werden.", utils.DefaultSendOptions())
		return err
	}

	if p.managerService.IsPluginDisabledForChat(c.EffectiveChat, pluginName) {
		_, err := c.EffectiveMessage.ReplyMessage(b, "💡 Plugin ist für diesen Chat schon deaktiviert", utils.DefaultSendOptions())
		return err
	}

	err := p.managerService.DisablePluginForChat(c.EffectiveChat, pluginName)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			_, err = c.EffectiveMessage.ReplyMessage(b, "❌ Plugin existiert nicht", utils.DefaultSendOptions())
			return err
		}

		guid := xid.New().String()
		log.Err(err).
			Str("guid", guid).
			Str("plugin", pluginName).
			Int64("chat_id", c.EffectiveChat.Id).
			Msg("Failed to disable plugin in chat")
		_, err := c.EffectiveMessage.ReplyMessage(b, fmt.Sprintf("❌ Es ist ein Fehler aufgetreten.%s", utils.EmbedGUID(guid)), utils.DefaultSendOptions())
		return err
	}

	return tgUtils.AddReactionWithFallback(b, c.EffectiveMessage, "👍", &tgUtils.ReactionFallbackOpts{
		Fallback: "✅ Plugin wurde für diesen Chat deaktiviert",
	})
}
