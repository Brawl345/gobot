package sql

import (
	"database/sql"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type cleverbotService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewCleverbotService(db *sqlx.DB) *cleverbotService {
	return &cleverbotService{
		DB:  db,
		log: logger.New("cleverbotService"),
	}
}

func (db *cleverbotService) SetState(chat *telebot.Chat, state string) error {
	const query = `UPDATE chats SET cleverbot_state = ? WHERE id = ?`
	_, err := db.Exec(query, state, chat.ID)
	return err
}

func (db *cleverbotService) ResetState(chat *telebot.Chat) error {
	const query = `UPDATE chats SET cleverbot_state = NULL WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *cleverbotService) GetState(chat *telebot.Chat) (string, error) {
	const query = `SELECT cleverbot_state FROM chats WHERE id = ?`
	var state sql.NullString
	err := db.Get(&state, query, chat.ID)
	return state.String, err
}
