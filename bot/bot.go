package bot

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/storage"
	"github.com/Brawl345/gobot/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

var log = logger.NewLogger("bot")

type (
	Nextbot struct {
		*telebot.Bot
		DB                     *storage.DB
		plugins                []IPlugin
		enabledPlugins         []string
		disabledPluginsForChat map[int64][]string
		allowedChats           []int64
	}

	IPlugin interface {
		GetName() string
		GetCommandHandlers() []CommandHandler
		GetCallbackHandlers() []CallbackHandler
		GetInlineHandlers() []InlineHandler
		Init()
	}

	CommandHandler struct {
		Command     any
		Handler     NextbotHandlerFunc
		AdminOnly   bool
		GroupOnly   bool
		HandleEdits bool
	}

	CallbackHandler struct {
		Command   *regexp.Regexp
		Handler   NextbotHandlerFunc
		AdminOnly bool
	}

	InlineHandler struct {
		Command             *regexp.Regexp
		Handler             NextbotHandlerFunc
		AdminOnly           bool
		CanBeUsedByEveryone bool
	}
)

func New() (*Nextbot, error) {
	db, err := storage.New()
	if err != nil {
		return nil, err
	}

	allowedUpdates := []string{"message", "edited_message", "callback_query", "inline_query"}

	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	webhookPort := strings.TrimSpace(os.Getenv("WEBHOOK_PORT"))
	webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_URL"))

	var poller telebot.Poller
	if webhookPort == "" || webhookURL == "" {
		log.Debug().Msg("Using long polling")
		poller = &telebot.LongPoller{
			AllowedUpdates: allowedUpdates,
			Timeout:        10 * time.Second,
		}
	} else {
		log.Debug().
			Str("port", webhookPort).
			Str("webhook_url", webhookURL).
			Msg("Using webhook")

		poller = &telebot.Webhook{
			Listen:         ":" + webhookPort,
			AllowedUpdates: allowedUpdates,
			MaxConnections: 50,
			DropUpdates:    true,
			Endpoint: &telebot.WebhookEndpoint{
				PublicURL: webhookURL,
			},
		}
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: poller,
	})

	if err != nil {
		return nil, err
	}

	// Calling "remove webook" even if no webhook is set
	// so pending updates can be dropped
	err = bot.RemoveWebhook(true)
	if err != nil {
		return nil, err
	}

	enabledPlugins, err := db.Plugins.GetAllEnabled()
	if err != nil {
		return nil, err
	}

	disabledPluginsForChat, err := db.ChatsPlugins.GetAllDisabled()
	if err != nil {
		return nil, err
	}

	allowedUsers, err := db.Users.GetAllAllowed()
	if err != nil {
		return nil, err
	}

	allowedChats, err := db.Chats.GetAllAllowed()
	if err != nil {
		return nil, err
	}

	allowedChats = append(allowedChats, allowedUsers...)

	return &Nextbot{
		Bot:                    bot,
		DB:                     db,
		enabledPlugins:         enabledPlugins,
		disabledPluginsForChat: disabledPluginsForChat,
		allowedChats:           allowedChats,
	}, nil
}

func (bot *Nextbot) RegisterPlugin(plugin IPlugin) {
	if plugin == nil {
		panic("plugin is nil")
	}
	plugin.Init()
	bot.plugins = append(bot.plugins, plugin)
}

func (bot *Nextbot) isPluginEnabled(pluginName string) bool {
	return slices.Contains(bot.enabledPlugins, pluginName)
}

func (bot *Nextbot) isPluginDisabledForChat(chat *telebot.Chat, pluginName string) bool {
	disabledPlugins, exists := bot.disabledPluginsForChat[chat.ID]
	if !exists {
		return false
	}
	return slices.Contains(disabledPlugins, pluginName)
}

func (bot *Nextbot) DisablePlugin(pluginName string) error {
	if !slices.Contains(bot.enabledPlugins, pluginName) {
		return errors.New("✅ Das Plugin ist nicht aktiv")
	}

	err := bot.DB.Plugins.Disable(pluginName)
	if err != nil {
		return err
	}
	index := slices.Index(bot.enabledPlugins, pluginName)
	bot.enabledPlugins = slices.Delete(bot.enabledPlugins, index, index+1)
	return nil
}

func (bot *Nextbot) DisablePluginForChat(chat *telebot.Chat, pluginName string) error {
	if bot.isPluginDisabledForChat(chat, pluginName) {
		return errors.New("✅ Das Plugin ist für diesen Chat schon deaktiviert")
	}

	for _, plugin := range bot.plugins {
		if plugin.GetName() == pluginName {
			err := bot.DB.ChatsPlugins.Disable(chat, pluginName)
			if err != nil {
				return err
			}

			bot.disabledPluginsForChat[chat.ID] = append(bot.disabledPluginsForChat[chat.ID], pluginName)

			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (bot *Nextbot) EnablePlugin(pluginName string) error {
	if slices.Contains(bot.enabledPlugins, pluginName) {
		return errors.New("✅ Das Plugin ist bereits aktiv")
	}

	for _, plugin := range bot.plugins {
		if plugin.GetName() == pluginName {
			err := bot.DB.Plugins.Enable(pluginName)
			if err != nil {
				return err
			}
			bot.enabledPlugins = append(bot.enabledPlugins, pluginName)
			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (bot *Nextbot) EnablePluginForChat(chat *telebot.Chat, pluginName string) error {
	if !bot.isPluginDisabledForChat(chat, pluginName) {
		return errors.New("✅ Das Plugin ist für diesen Chat schon aktiv")
	}

	for _, plugin := range bot.plugins {
		if plugin.GetName() == pluginName {
			err := bot.DB.ChatsPlugins.Enable(chat, pluginName)
			if err != nil {
				return err
			}

			index := slices.Index(bot.disabledPluginsForChat[chat.ID], pluginName)
			bot.disabledPluginsForChat[chat.ID] = slices.Delete(bot.disabledPluginsForChat[chat.ID],
				index, index+1)

			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (bot *Nextbot) IsUserAllowed(user *telebot.User) bool {
	if utils.IsAdmin(user) {
		return true
	}

	return slices.Contains(bot.allowedChats, user.ID)
}

func (bot *Nextbot) IsChatAllowed(chat *telebot.Chat) bool {
	return slices.Contains(bot.allowedChats, chat.ID)
}

func (bot *Nextbot) AllowUser(user *telebot.User) error {
	err := bot.DB.Users.Allow(user)
	if err != nil {
		return err
	}

	bot.allowedChats = append(bot.allowedChats, user.ID)
	return nil
}

func (bot *Nextbot) DenyUser(user *telebot.User) error {
	if utils.IsAdmin(user) {
		return errors.New("cannot deny admin")
	}
	err := bot.DB.Users.Deny(user)
	if err != nil {
		return err
	}

	index := slices.Index(bot.allowedChats, user.ID)
	bot.allowedChats = slices.Delete(bot.allowedChats, index, index+1)
	return nil
}

func (bot *Nextbot) AllowChat(chat *telebot.Chat) error {
	err := bot.DB.Chats.Allow(chat)
	if err != nil {
		return err
	}

	bot.allowedChats = append(bot.allowedChats, chat.ID)
	return nil
}

func (bot *Nextbot) DenyChat(chat *telebot.Chat) error {
	err := bot.DB.Chats.Deny(chat)
	if err != nil {
		return err
	}

	index := slices.Index(bot.allowedChats, chat.ID)
	bot.allowedChats = slices.Delete(bot.allowedChats, index, index+1)
	return nil
}
