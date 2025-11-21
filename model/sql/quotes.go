package sql

import (
	"database/sql"
	"errors"
	"math/rand"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
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

func (db *quoteService) GetQuote(chat *gotgbot.Chat) (string, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM quotes WHERE chat_id = ?", chat.Id)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", model.ErrNotFound
	}

	offset := rand.Intn(count)
	var quote string
	err = db.Get(&quote, "SELECT quote FROM quotes WHERE chat_id = ? LIMIT 1 OFFSET ?", chat.Id, offset)
	if errors.Is(err, sql.ErrNoRows) {
		return "", model.ErrNotFound
	}
	return quote, err
}

func (db *quoteService) exists(chat *gotgbot.Chat, quote string) (bool, error) {
	const query = `SELECT 1 FROM quotes WHERE chat_id = ? AND quote = ?`
	var exists bool
	err := db.Get(&exists, query, chat.Id, quote)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

func (db *quoteService) SaveQuote(chat *gotgbot.Chat, quote string) error {
	exists, err := db.exists(chat, quote)
	if err != nil {
		return err
	}
	if exists {
		return model.ErrAlreadyExists
	}

	const query = `INSERT INTO quotes (chat_id, quote) VALUES (?, ?)`

	_, err = db.Exec(query, chat.Id, quote)
	return err
}

func (db *quoteService) DeleteQuote(chat *gotgbot.Chat, quote string) error {
	exists, err := db.exists(chat, quote)
	if err != nil {
		return err
	}
	if !exists {
		return model.ErrNotFound
	}

	const query = `DELETE FROM quotes WHERE chat_id = ? AND quote = ?`

	_, err = db.Exec(query, chat.Id, quote)
	return err
}
