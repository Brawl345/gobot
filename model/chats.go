package model

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/jmoiron/sqlx"
)

type ChatService interface {
	Allow(chat *gotgbot.Chat) error
	Create(chat *gotgbot.Chat) error
	CreateTx(tx *sqlx.Tx, chat *gotgbot.Chat) error
	Deny(chat *gotgbot.Chat) error
	GetAllAllowed() ([]int64, error)
}
