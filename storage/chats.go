package storage

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	ChatStorage interface {
		Create(chat *telebot.Chat) error
		CreateWithTx(tx *sqlx.Tx, chat *telebot.Chat) error
	}

	Chats struct {
		*sqlx.DB
	}
)

func (db *Chats) Create(chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := db.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *Chats) CreateWithTx(tx *sqlx.Tx, chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := tx.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}
