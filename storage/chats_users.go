package storage

import (
	"context"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	ChatUserStorage interface {
		Create(chat *telebot.Chat, user *telebot.User) error
		CreateBatch(chat *telebot.Chat, users *[]telebot.User) error
		GetAllUsersWithMsgCount(chat *telebot.Chat) ([]User, error)
		IsAllowed(chat *telebot.Chat, user *telebot.User) bool
		Leave(chat *telebot.Chat, user *telebot.User) error
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

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	err = db.Users.CreateTx(tx, user)
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

func (db *ChatsUsers) CreateBatch(chat *telebot.Chat, users *[]telebot.User) error {
	const insertRelationshipQuery = `INSERT INTO 
    chats_users (chat_id, user_id, msg_count, in_group) 
    VALUES (?, ?, 0, true)
    ON DUPLICATE KEY UPDATE chat_id = chat_id, in_group = true`

	tx, err := db.BeginTxx(context.Background(), nil)

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	// creating a query for every user is inefficient,
	// but idc
	for _, user := range *users {
		if user.IsBot {
			continue
		}

		err = db.Users.CreateTx(tx, &user)
		if err != nil {
			return err
		}

		_, err := tx.Exec(insertRelationshipQuery, chat.ID, user.ID)
		if err != nil {
			return err
		}
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

func (db *ChatsUsers) IsAllowed(chat *telebot.Chat, user *telebot.User) bool {
	if isAdmin(user) {
		return true
	}

	const query = `SELECT 1 FROM chats, users 
	WHERE chats.id = ?
	AND chats.allowed = true
	OR (users.id = ? AND users.allowed = true);`

	var isAllowed bool
	db.Get(&isAllowed, query, chat.ID, user.ID)
	return isAllowed
}

func (db *ChatsUsers) Leave(chat *telebot.Chat, user *telebot.User) error {
	const query = `UPDATE chats_users SET in_group = false
	WHERE chat_id = ?
	  AND user_id = ?`

	_, err := db.Exec(query, chat.ID, user.ID)
	return err
}
