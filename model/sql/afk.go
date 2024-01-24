package sql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
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

func (db *afkService) SetAFK(chat *gotgbot.Chat, user *gotgbot.Sender, now time.Time) error {
	const query = `UPDATE chats_users
	SET afk_since = ?,
	    afk_reason = NULL
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(query, now, chat.Id, user.Id())
	return err
}

func (db *afkService) SetAFKWithReason(chat *gotgbot.Chat, user *gotgbot.Sender, reason string) error {
	const query = `UPDATE chats_users
	SET afk_since = CURRENT_TIME(),
	    afk_reason = ?
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(query, reason, chat.Id, user.Id())
	return err
}

func (db *afkService) IsAFK(chat *gotgbot.Chat, user *gotgbot.Sender) (bool, model.AFKData, error) {
	const query = `SELECT afk_since, afk_reason
	FROM chats_users
	WHERE chat_id = ?
	  AND user_id = ?
	  AND afk_since IS NOT NULL`
	var afkData model.AFKData
	err := db.Get(&afkData, query, chat.Id, user.Id())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, afkData, nil
		}
		return false, afkData, err
	}
	return true, afkData, nil
}

func (db *afkService) BackAgain(chat *gotgbot.Chat, user *gotgbot.Sender) error {
	const updateQuery = `UPDATE chats_users
	SET afk_since = NULL,
	    afk_reason = NULL	
	WHERE chat_id = ?
	  AND user_id = ?`
	_, err := db.Exec(updateQuery, chat.Id, user.Id())
	return err
}

func (db *afkService) IsAFKByUsername(chat *gotgbot.Chat, username string) (bool, model.AFKData, error) {
	const query = `SELECT afk_since, afk_reason, first_name
	FROM chats_users
	LEFT JOIN users ON chats_users.user_id = users.id
	WHERE chat_id = ?
	  AND in_group = TRUE
	  AND username = ?
	  AND afk_since IS NOT NULL`

	var afkData model.AFKData
	err := db.Get(&afkData, query, chat.Id, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, afkData, nil
		}
		return false, afkData, err
	}
	return true, afkData, nil
}
