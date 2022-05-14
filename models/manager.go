package models

import (
	"github.com/Brawl345/gobot/plugin"
	"gopkg.in/telebot.v3"
)

type ManagerService interface {
	Plugins() []plugin.Plugin
	EnablePlugin(name string) error
	EnablePluginForChat(chat *telebot.Chat, name string) error
	DisablePlugin(name string) error
	DisablePluginForChat(chat *telebot.Chat, name string) error
	IsPluginEnabled(name string) bool
	IsPluginDisabledForChat(chat *telebot.Chat, name string) bool
}
