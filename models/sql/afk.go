package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type afkService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewAfkService(db *sqlx.DB) *afkService {
	return &afkService{
		DB:  db,
		log: logger.New("afkService"),
	}
}

func (db *afkService) SetAFK(chat *telebot.Chat, user *telebot.User) error {
	const query = `UPDATE chats_users
	SET afk_since = CURRENT_TIME(),
	    afk_reason = NULL
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(query, chat.ID, user.ID)
	return err
}

func (db *afkService) SetAFKWithReason(chat *telebot.Chat, user *telebot.User, reason string) error {
	const query = `UPDATE chats_users
	SET afk_since = CURRENT_TIME(),
	    afk_reason = ?
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(query, reason, chat.ID, user.ID)
	return err
}

func (db *afkService) IsAFK(chat *telebot.Chat, user *telebot.User) (bool, models.AFKData, error) {
	const query = `SELECT afk_since, afk_reason
	FROM chats_users
	WHERE chat_id = ?
	  AND user_id = ?
	  AND afk_since IS NOT NULL`
	var afkData models.AFKData
	err := db.Get(&afkData, query, chat.ID, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, afkData, nil
		}
		return false, afkData, err
	}
	return true, afkData, nil
}

func (db *afkService) BackAgain(chat *telebot.Chat, user *telebot.User) error {
	const updateQuery = `UPDATE chats_users
	SET afk_since = NULL,
	    afk_reason = NULL	
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(updateQuery, chat.ID, user.ID)
	return err
}
