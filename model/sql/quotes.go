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

func (db *quoteService) SaveQuote(chat *gotgbot.Chat, quote string) error {
	const query = `INSERT INTO quotes (chat_id, quote)
	SELECT ?, ? FROM DUAL
	WHERE NOT EXISTS (SELECT 1 FROM quotes WHERE chat_id = ? AND quote = ?)`

	res, err := db.Exec(query, chat.Id, quote, chat.Id, quote)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrAlreadyExists
	}
	return nil
}

func (db *quoteService) DeleteQuote(chat *gotgbot.Chat, quote string) error {
	const query = `DELETE FROM quotes WHERE chat_id = ? AND quote = ?`

	res, err := db.Exec(query, chat.Id, quote)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}
