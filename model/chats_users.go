package model

import (
	"gopkg.in/telebot.v3"
)

type ChatsUsersService interface {
	Create(chat *telebot.Chat, user *telebot.User) error
	CreateBatch(chat *telebot.Chat, users *[]telebot.User) error
	GetAllUsersWithMsgCount(chat *telebot.Chat) ([]User, error)
	IsAllowed(chat *telebot.Chat, user *telebot.User) bool
	Leave(chat *telebot.Chat, user *telebot.User) error
}
