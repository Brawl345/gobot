package storage

import (
	"context"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	ChatUserStorage interface {
		Create(chat *telebot.Chat, user *telebot.User) error
		GetAllUsersWithMsgCount(chat *telebot.Chat) ([]User, error)
		IsAllowed(chat *telebot.Chat, user *telebot.User) (bool, error)
	}

	ChatsUsers struct {
		Chats ChatStorage
		Users UserStorage
		*sqlx.DB
	}
)

func (db *ChatsUsers) Create(chat *telebot.Chat, user *telebot.User) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	err = db.Chats.CreateWithTx(tx, chat)
	if err != nil {
		return err
	}

	err = db.Users.CreateWithTx(tx, user)
	if err != nil {
		return err
	}

	err = db.insertRelationship(tx, chat.ID, user.ID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *ChatsUsers) GetAllUsersWithMsgCount(chat *telebot.Chat) ([]User, error) {
	const query = `SELECT u.first_name, u.last_name, msg_count FROM chats_users
JOIN users u on u.id = chats_users.user_id
WHERE chat_id = ?
ORDER BY msg_count DESC`
	var users []User
	err := db.Select(&users, query, chat.ID)
	return users, err
}

func (db *ChatsUsers) insertRelationship(tx *sqlx.Tx, chatId int64, userId int64) error {
	const query = `INSERT INTO 
    chats_users (chat_id, user_id, in_group) 
    VALUES (?, ?, true)
    ON DUPLICATE KEY UPDATE chat_id = chat_id, msg_count = msg_count + 1, in_group = true`
	_, err := tx.Exec(query, chatId, userId)
	return err
}

func (db *ChatsUsers) IsAllowed(chat *telebot.Chat, user *telebot.User) (bool, error) {
	if isAdmin(user) {
		return true, nil
	}

	const query = `SELECT 1 FROM chats_users
    JOIN chats ON chats_users.chat_id = chats.id
    JOIN users ON chats_users.user_id = users.id
WHERE chats_users.chat_id = ?
  AND chats_users.user_id = ?
  AND (chats.allowed = true
           OR users.allowed = true)`

	var isAllowed bool
	err := db.Get(&isAllowed, query, chat.ID, user.ID)
	return isAllowed, err
}
