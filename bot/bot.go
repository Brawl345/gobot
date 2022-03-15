package bot

import (
	"errors"
	"github.com/Brawl345/gobot/storage"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
	"log"
	"time"
)

// TODO: Disabled plugins für chat
type Nextbot struct {
	*telebot.Bot
	DB             *storage.DB
	plugins        []IPlugin
	enabledPlugins []string
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

	return &Nextbot{
		Bot:            bot,
		DB:             db,
		enabledPlugins: enabledPlugins,
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
	for _, enabledPlugin := range bot.enabledPlugins {
		if enabledPlugin == pluginName {
			return true
		}
	}
	log.Printf("Plugin %s is disabled globally", pluginName)
	return false
}

func (bot *Nextbot) DisablePlugin(pluginName string) error {
	if !slices.Contains(bot.enabledPlugins, pluginName) {
		return errors.New("✅ Das Plugin ist bereits deaktiviert")
	}

	for _, plugin := range bot.plugins {
		if plugin.GetName() == pluginName {
			err := bot.DB.Plugins.Disable(pluginName)
			if err != nil {
				return err
			}
			index := slices.Index(bot.enabledPlugins, pluginName)
			bot.enabledPlugins = slices.Delete(bot.enabledPlugins, index, index+1)
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

func (bot *Nextbot) OnText(c telebot.Context) error {
	var err error

	if c.Message().Private() {
		err = bot.DB.Users.Create(c.Sender())
	} else {
		err = bot.DB.ChatsUsers.Create(c.Chat(), c.Sender())
	}
	if err != nil {
		return err
	}
	log.Printf("%s: %s", c.Chat().FirstName, c.Message().Text)

	var isAllowed bool
	if c.Message().Private() {
		isAllowed, _ = bot.DB.Users.IsAllowed(c.Sender())
	} else {
		isAllowed, _ = bot.DB.ChatsUsers.IsAllowed(c.Chat(), c.Sender())
	}

	if !isAllowed {
		return nil
	}

	text := c.Message().Caption
	if text == "" {
		text = c.Message().Text
	}

	for _, plugin := range bot.plugins {
		for _, handler := range plugin.GetHandlers() {
			matches := handler.Command.FindStringSubmatch(text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plugin.GetName(), handler.Command)
				if bot.isPluginEnabled(plugin.GetName()) {
					ctx := NextbotContext{
						Context: c,
						Matches: matches,
					}
					go handler.Handler(ctx)
				}
			}
		}
	}

	return nil
}
