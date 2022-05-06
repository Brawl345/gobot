package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type quoteService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewQuoteService(db *sqlx.DB) *quoteService {
	return &quoteService{
		DB:  db,
		log: logger.New("quoteService"),
	}
}

func (db *quoteService) GetQuote(chat *telebot.Chat) (string, error) {
	var quote string
	err := db.Get(&quote, "SELECT quote FROM quotes WHERE chat_id = ? ORDER BY RAND() LIMIT 1", chat.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", models.ErrNotFound
	}
	return quote, err
}

func (db *quoteService) exists(chat *telebot.Chat, quote string) (bool, error) {
	const query = `SELECT 1 FROM quotes WHERE chat_id = ? AND quote = ?`
	var exists bool
	err := db.Get(&exists, query, chat.ID, quote)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

func (db *quoteService) SaveQuote(chat *telebot.Chat, quote string) error {
	exists, err := db.exists(chat, quote)
	if err != nil {
		return err
	}
	if exists {
		return models.ErrAlreadyExists
	}

	const query = `INSERT INTO quotes (chat_id, quote) VALUES (?, ?)`

	_, err = db.Exec(query, chat.ID, quote)
	return err
}

func (db *quoteService) DeleteQuote(chat *telebot.Chat, quote string) error {
	exists, err := db.exists(chat, quote)
	if err != nil {
		return err
	}
	if !exists {
		return models.ErrNotFound
	}

	const query = `DELETE FROM quotes WHERE chat_id = ? AND quote = ?`

	_, err = db.Exec(query, chat.ID, quote)
	return err
}
