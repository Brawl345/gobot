package sql

import (
	"database/sql"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
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

func (db *cleverbotService) SetState(chat *gotgbot.Chat, state string) error {
	const query = `UPDATE chats SET cleverbot_state = $1 WHERE id = $2`
	_, err := db.Exec(query, state, chat.Id)
	return err
}

func (db *cleverbotService) ResetState(chat *gotgbot.Chat) error {
	const query = `UPDATE chats SET cleverbot_state = NULL WHERE id = $1`
	_, err := db.Exec(query, chat.Id)
	return err
}

func (db *cleverbotService) GetState(chat *gotgbot.Chat) (string, error) {
	const query = `SELECT cleverbot_state FROM chats WHERE id = $1`
	var state sql.NullString
	err := db.Get(&state, query, chat.Id)
	return state.String, err
}
