package model

import (
	"github.com/Brawl345/gobot/plugin"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type ManagerService interface {
	Plugins() []plugin.Plugin
	EnablePlugin(name string) error
	EnablePluginForChat(chat *gotgbot.Chat, name string) error
	DisablePlugin(name string) error
	DisablePluginForChat(chat *gotgbot.Chat, name string) error
	IsPluginEnabled(name string) bool
	IsPluginDisabledForChat(chat *gotgbot.Chat, name string) bool
}
