package model

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type ChatsUsersService interface {
	Create(chat *gotgbot.Chat, user *gotgbot.User) error
	CreateBatch(chat *gotgbot.Chat, users *[]gotgbot.User) error
	GetAllUsersWithMsgCount(chat *gotgbot.Chat) ([]User, error)
	IsAllowed(chat *gotgbot.Chat, user *gotgbot.User) bool
	Leave(chat *gotgbot.Chat, user *gotgbot.User) error
}
