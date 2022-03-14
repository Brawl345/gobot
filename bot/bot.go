package bot

import (
	"github.com/Brawl345/gobot/storage"
	"gopkg.in/telebot.v3"
	"log"
	"time"
)

type Nextbot struct {
	*telebot.Bot
	DB      *storage.DB
	plugins []IPlugin
}

func NewBot(token string, db *storage.DB) (*Nextbot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, err
	}

	return &Nextbot{
		Bot: bot,
		DB:  db,
	}, nil
}

func (bot *Nextbot) RegisterPlugin(plugin IPlugin) {
	plugin.Init()
	bot.plugins = append(bot.plugins, plugin)
}

func (bot *Nextbot) isPluginDisabled(pluginName string) bool {
	return false
}

func (bot *Nextbot) OnText(c telebot.Context) error {
	log.Printf("%s: %s", c.Chat().FirstName, c.Message().Text)

	for _, plugin := range bot.plugins {
		for _, handler := range plugin.GetHandlers() {
			if handler.Command.MatchString(c.Message().Text) {
				log.Printf("Matched command %s by %s", handler.Command, plugin.GetName())
				if !bot.isPluginDisabled(plugin.GetName()) {
					go handler.Handler(c)
				}
			}
		}
	}

	return nil
}
