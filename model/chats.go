package model

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type ChatService interface {
	Allow(chat *telebot.Chat) error
	Create(chat *telebot.Chat) error
	CreateTx(tx *sqlx.Tx, chat *telebot.Chat) error
	Deny(chat *telebot.Chat) error
	GetAllAllowed() ([]int64, error)
}
