package sql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
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

func (db *afkService) SetAFK(chat *telebot.Chat, user *telebot.User, now time.Time) error {
	const query = `UPDATE chats_users
	SET afk_since = ?,
	    afk_reason = NULL
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(query, now, chat.ID, user.ID)
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

func (db *afkService) IsAFK(chat *telebot.Chat, user *telebot.User) (bool, model.AFKData, error) {
	const query = `SELECT afk_since, afk_reason
	FROM chats_users
	WHERE chat_id = ?
	  AND user_id = ?
	  AND afk_since IS NOT NULL`
	var afkData model.AFKData
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

func (db *afkService) IsAFKByUsername(chat *telebot.Chat, username string) (bool, model.AFKData, error) {
	const query = `SELECT afk_since, afk_reason, first_name
	FROM chats_users
	LEFT JOIN users ON chats_users.user_id = users.id
	WHERE chat_id = ?
	  AND in_group = TRUE
	  AND username = ?
	  AND afk_since IS NOT NULL`

	var afkData model.AFKData
	err := db.Get(&afkData, query, chat.ID, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, afkData, nil
		}
		return false, afkData, err
	}
	return true, afkData, nil
}
