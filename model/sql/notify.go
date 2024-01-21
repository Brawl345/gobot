package sql

import (
	"database/sql"
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type notifyService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewNotifyService(db *sqlx.DB) *notifyService {
	return &notifyService{
		DB:  db,
		log: logger.New("notifyService"),
	}
}

func (db *notifyService) Enabled(chat *gotgbot.Chat, user *gotgbot.User) (bool, error) {
	const query = `SELECT notify
	FROM chats_users
	WHERE chat_id = ?
	  AND user_id = ?;`
	var enabled bool
	err := db.Get(&enabled, query, chat.Id, user.Id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return enabled, err
}

func (db *notifyService) Enable(chat *gotgbot.Chat, user *gotgbot.User) error {
	const query = `UPDATE chats_users
	SET notify = true
	WHERE chat_id = ?
	  AND user_id = ?;`
	_, err := db.Exec(query, chat.Id, user.Id)
	return err
}

func (db *notifyService) Disable(chat *gotgbot.Chat, user *gotgbot.User) error {
	const query = `UPDATE chats_users
	SET notify = false
	WHERE chat_id = ?
	  AND user_id = ?;`
	_, err := db.Exec(query, chat.Id, user.Id)
	return err
}

func (db *notifyService) GetAllToBeNotifiedUsers(chat *gotgbot.Chat, mentionedUsernames []string) ([]int64, error) {
	query := `SELECT u.id FROM chats_users cu
	LEFT JOIN users u ON cu.user_id = u.id
	WHERE cu.notify = TRUE
	  AND cu.chat_id = ?
	  AND cu.in_group = TRUE
	  AND u.username IN (?);`
	query, args, err := sqlx.In(query, chat.Id, mentionedUsernames)
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
