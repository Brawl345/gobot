package bot

import (
	"errors"
	"github.com/Brawl345/gobot/storage"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
	"time"
)

type Nextbot struct {
	*telebot.Bot
	DB                     *storage.DB
	plugins                []IPlugin
	enabledPlugins         []string
	disabledPluginsForChat map[int64][]string
	allowedChats           []int64
}

func NewBot(token string, db *storage.DB) (*Nextbot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token: token,
		Poller: &telebot.LongPoller{
			AllowedUpdates: []string{"message", "callback_query"}, // TODO: Inline Query
			Timeout:        10 * time.Second,
		},
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
	if isAdmin(user) {
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
	if isAdmin(user) {
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
