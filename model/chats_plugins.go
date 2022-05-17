package model

import (
	"gopkg.in/telebot.v3"
)

type ChatsPluginsService interface {
	Disable(chat *telebot.Chat, pluginName string) error
	Enable(chat *telebot.Chat, pluginName string) error
	GetAllDisabled() (map[int64][]string, error)
}
