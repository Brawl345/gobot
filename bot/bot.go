package bot

import (
	"errors"
	"github.com/Brawl345/gobot/storage"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
	"log"
	"time"
)

type Nextbot struct {
	*telebot.Bot
	DB                     *storage.DB
	plugins                []IPlugin
	enabledPlugins         []string
	disabledPluginsForChat map[int64][]string
}

func NewBot(token string, db *storage.DB) (*Nextbot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token: token,
		Poller: &telebot.LongPoller{
			AllowedUpdates: []string{"message"}, // TODO: Callback & Inline
			Timeout:        10 * time.Second,
		},
	})

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

	return &Nextbot{
		Bot:                    bot,
		DB:                     db,
		enabledPlugins:         enabledPlugins,
		disabledPluginsForChat: disabledPluginsForChat,
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

func (bot *Nextbot) OnText(c telebot.Context) error {
	log.Printf("%s: %s", c.Chat().FirstName, c.Message().Text)

	var isAllowed bool
	// TODO: Allow-Liste cachen?
	if c.Message().Private() {
		isAllowed = bot.DB.Users.IsAllowed(c.Sender())
	} else {
		isAllowed = bot.DB.ChatsUsers.IsAllowed(c.Chat(), c.Sender())
	}

	if !isAllowed {
		return nil
	}

	var err error

	if c.Message().Private() {
		err = bot.DB.Users.Create(c.Sender())
	} else {
		err = bot.DB.ChatsUsers.Create(c.Chat(), c.Sender())
	}
	if err != nil {
		return err
	}

	text := c.Message().Caption
	if text == "" {
		text = c.Message().Text
	}

	for _, plugin := range bot.plugins {
		for _, handler := range plugin.GetHandlers() {
			if !c.Message().FromGroup() && handler.GroupOnly {
				continue
			}

			matches := handler.Command.FindStringSubmatch(text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plugin.GetName(), handler.Command)
				if bot.isPluginEnabled(plugin.GetName()) {
					if c.Message().FromGroup() && bot.isPluginDisabledForChat(c.Chat(), plugin.GetName()) {
						log.Printf("Plugin %s is disabled for this chat", plugin.GetName())
						continue
					}

					if handler.AdminOnly && !isAdmin(c.Sender()) {
						log.Println("User is not an admin.")
						continue
					}

					go handler.Handler(NextbotContext{
						Context: c,
						Matches: matches,
					})
				} else {
					log.Printf("Plugin %s is disabled globally", plugin.GetName())
				}
			}
		}
	}

	return nil
}
