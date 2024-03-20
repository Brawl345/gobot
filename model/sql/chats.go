package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/jmoiron/sqlx"
)

type chatService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewChatService(db *sqlx.DB) *chatService {
	return &chatService{
		DB:  db,
		log: logger.New("chatService"),
	}
}

func (db *chatService) Allow(chat *gotgbot.Chat) error {
	const query = `UPDATE chats SET allowed = true WHERE id = $1`
	_, err := db.Exec(query, chat.Id)
	return err
}

func (db *chatService) Create(chat *gotgbot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES ($1, $2)
    ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title`
	_, err := db.Exec(query, chat.Id, chat.Title)
	return err
}

func (db *chatService) CreateTx(tx *sqlx.Tx, chat *gotgbot.Chat) error {
	const query = `INSERT INTO 
    chats (id, title)
    VALUES ($1, $2)
    ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title`
	_, err := tx.Exec(query, chat.Id, chat.Title)
	return err
}

func (db *chatService) Deny(chat *gotgbot.Chat) error {
	const query = `UPDATE chats SET allowed = false WHERE id = $1`
	_, err := db.Exec(query, chat.Id)
	return err
}

func (db *chatService) GetAllAllowed() ([]int64, error) {
	const query = `SELECT id FROM chats WHERE allowed = true`

	var allowed []int64
	err := db.Select(&allowed, query)

	return allowed, err
}
