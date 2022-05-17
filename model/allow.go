package model

import "gopkg.in/telebot.v3"

type AllowService interface {
	AllowChat(chat *telebot.Chat) error
	AllowUser(user *telebot.User) error
	DenyChat(chat *telebot.Chat) error
	DenyUser(user *telebot.User) error
	IsChatAllowed(chat *telebot.Chat) bool
	IsUserAllowed(user *telebot.User) bool
}
