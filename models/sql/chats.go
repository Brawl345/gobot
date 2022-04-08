package sql

import (
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type ChatService struct {
	*sqlx.DB
}

func NewChatService(db *sqlx.DB) *ChatService {
	return &ChatService{db}
}

func (db *ChatService) Allow(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *ChatService) Create(chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := db.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *ChatService) CreateTx(tx *sqlx.Tx, chat *telebot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES (? ,?)
    ON DUPLICATE KEY UPDATE title = ?`
	_, err := tx.Exec(query, chat.ID, chat.Title, chat.Title)
	return err
}

func (db *ChatService) Deny(chat *telebot.Chat) error {
	const query = `UPDATE chats SET allowed = false WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *ChatService) GetAllAllowed() ([]int64, error) {
	const query = `SELECT id FROM chats WHERE allowed = true`

	var allowed []int64
	err := db.Select(&allowed, query)

	return allowed, err
}
