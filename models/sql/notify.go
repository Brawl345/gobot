package sql

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type notifyService struct {
	*sqlx.DB
}

func NewNotifyService(db *sqlx.DB) *notifyService {
	return &notifyService{db}
}

func (db *notifyService) Enabled(chat *telebot.Chat, user *telebot.User) (bool, error) {
	const query = `SELECT notify
	FROM chats_users
	WHERE chat_id = ?
	  AND user_id = ?;`
	var enabled bool
	err := db.Get(&enabled, query, chat.ID, user.ID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return enabled, err
}

func (db *notifyService) Enable(chat *telebot.Chat, user *telebot.User) error {
	const query = `UPDATE chats_users
	SET notify = true
	WHERE chat_id = ?
	  AND user_id = ?;`
	_, err := db.Exec(query, chat.ID, user.ID)
	return err
}

func (db *notifyService) Disable(chat *telebot.Chat, user *telebot.User) error {
	const query = `UPDATE chats_users
	SET notify = false
	WHERE chat_id = ?
	  AND user_id = ?;`
	_, err := db.Exec(query, chat.ID, user.ID)
	return err
}

func (db *notifyService) GetAllToBeNotifiedUsers(chat *telebot.Chat, mentionedUsernames []string) ([]int64, error) {
	query := `SELECT u.id FROM chats_users cu
	LEFT JOIN users u ON cu.user_id = u.id
	WHERE cu.notify = TRUE
	  AND cu.chat_id = ?
	  AND cu.in_group = TRUE
	  AND u.username IN (?);`
	query, args, err := sqlx.In(query, chat.ID, mentionedUsernames)
	if err != nil {
		return nil, err
	}
	query = db.Rebind(query)

	var userIDs []int64
	err = db.Select(&userIDs, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return userIDs, nil
}
