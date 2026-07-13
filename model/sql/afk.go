package sql

import (
	"database/sql"
	"errors"
	"strings"
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

// AFKByUsernames returns the AFK data for every given username that is
// currently AFK in the chat, keyed by lowercased username.
func (db *afkService) AFKByUsernames(chat *gotgbot.Chat, usernames []string) (map[string]model.AFKData, error) {
	result := make(map[string]model.AFKData)
	if len(usernames) == 0 {
		return result, nil
	}

	const query = `SELECT afk_since, afk_reason, first_name, username
	FROM chats_users
	LEFT JOIN users ON chats_users.user_id = users.id
	WHERE chat_id = ?
	  AND in_group = TRUE
	  AND username IN (?)
	  AND afk_since IS NOT NULL`

	q, args, err := sqlx.In(query, chat.Id, usernames)
	if err != nil {
		return nil, err
	}
	q = db.Rebind(q)

	rows, err := db.Queryx(q, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	for rows.Next() {
		var row struct {
			model.AFKData
			Username string `db:"username"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		result[strings.ToLower(row.Username)] = row.AFKData
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
