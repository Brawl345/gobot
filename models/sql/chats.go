package sql

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type chatService struct {
	*sqlx.DB
}

func NewChatService(db *sqlx.DB) *chatService {
	return &chatService{db}
}

func (db *chatService) Allow(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *chatService) Create(chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := db.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *chatService) CreateTx(tx *sqlx.Tx, chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := tx.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *chatService) Deny(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = false WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *chatService) GetAllAllowed() ([]int64, error) {
	const query = `SELECT id FROM chats WHERE allowed = true`

	var allowed []int64
	err := db.Select(&allowed, query)

	return allowed, err
}
