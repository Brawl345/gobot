package sql

import (
	"database/sql"
	"errors"
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
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

func (db *geminiService) GetHistory(chat *telebot.Chat) (model.GeminiData, error) {
	const query = `SELECT gemini_history, gemini_history_expires_on FROM chats WHERE id = ?`
	var geminiData model.GeminiData
	err := db.Get(&geminiData, query, chat.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return geminiData, nil
		}
	}
	return geminiData, err
}

func (db *geminiService) ResetHistory(chat *telebot.Chat) error {
	const query = `UPDATE chats SET gemini_history = NULL, gemini_history_expires_on = NULL WHERE id = ?`
	_, err := db.Exec(query, chat.ID)
	return err
}

func (db *geminiService) SetHistory(chat *telebot.Chat, history string) error {
	const query = `UPDATE chats
	SET gemini_history = ?,
	    gemini_history_expires_on = NOW() + INTERVAL 10 MINUTE 
	WHERE id = ?`
	_, err := db.Exec(query, history, chat.ID)
	return err
}
