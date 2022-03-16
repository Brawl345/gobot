package storage

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	ChatStorage interface {
		Allow(chat *telebot.Chat) error
		Create(chat *telebot.Chat) error
		CreateTx(tx *sqlx.Tx, chat *telebot.Chat) error
		Deny(chat *telebot.Chat) error
		IsAllowed(chat *telebot.Chat) bool
	}

	Chats struct {
		*sqlx.DB
	}
)

func (db *Chats) Allow(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *Chats) Create(chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := db.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *Chats) CreateTx(tx *sqlx.Tx, chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := tx.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *Chats) Deny(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = false WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *Chats) IsAllowed(chat *telebot.Chat) bool {
	const query = `SELECT allowed FROM chats WHERE id = ?`

	var isAllowed bool
	db.Get(&isAllowed, query, chat.ID)
	return isAllowed
}
