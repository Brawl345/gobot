package model

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type ChatsPluginsService interface {
	Disable(chat *gotgbot.Chat, pluginName string) error
	Enable(chat *gotgbot.Chat, pluginName string) error
	GetAllDisabled() (map[int64][]string, error)
}
