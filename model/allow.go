package model

import "github.com/PaulSonOfLars/gotgbot/v2"

type AllowService interface {
	AllowChat(chat *gotgbot.Chat) error
	AllowUser(user *gotgbot.User) error
	DenyChat(chat *gotgbot.Chat) error
	DenyUser(user *gotgbot.User) error
	IsChatAllowed(chat *gotgbot.Chat) bool
	IsUserAllowed(user *gotgbot.User) bool
}
