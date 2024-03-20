package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/jmoiron/sqlx"
)

type (
	geminiService struct {
		*sqlx.DB
		log *logger.Logger
	}
)

func NewGeminiService(db *sqlx.DB) *geminiService {
	return &geminiService{
		DB:  db,
		log: logger.New("geminiService"),
	}
}

func (db *geminiService) GetHistory(chat *gotgbot.Chat) (model.GeminiData, error) {
	const query = `SELECT gemini_history, gemini_history_expires_on FROM chats WHERE id = $1`
	var geminiData model.GeminiData
	err := db.Get(&geminiData, query, chat.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return geminiData, nil
		}
	}
	return geminiData, err
}

func (db *geminiService) ResetHistory(chat *gotgbot.Chat) error {
	const query = `UPDATE chats SET gemini_history = NULL, gemini_history_expires_on = NULL WHERE id = $1`
	_, err := db.Exec(query, chat.Id)
	return err
}

func (db *geminiService) SetHistory(chat *gotgbot.Chat, history string) error {
	const query = `UPDATE chats
	SET gemini_history = $1,
	    gemini_history_expires_on = NOW() + INTERVAL '10 MINUTE' 
	WHERE id = $2`
	_, err := db.Exec(query, history, chat.Id)
	return err
}
