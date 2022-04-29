package sql

import (
	"context"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/utils"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type chatsUsersService struct {
	Chats models.ChatService
	Users models.UserService
	*sqlx.DB
}

func NewChatsUsersService(db *sqlx.DB, chatService models.ChatService, userService models.UserService) *chatsUsersService {
	return &chatsUsersService{
		Chats: chatService,
		Users: userService,
		DB:    db,
	}
}

func (db *chatsUsersService) Create(chat *telebot.Chat, user *telebot.User) error {
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

func (db *chatsUsersService) CreateBatch(chat *telebot.Chat, users *[]telebot.User) error {
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

func (db *chatsUsersService) GetAllUsersWithMsgCount(chat *telebot.Chat) ([]models.User, error) {
	const query = `SELECT u.first_name, u.last_name, msg_count, in_group FROM chats_users
JOIN users u on u.id = chats_users.user_id
WHERE chat_id = ?
ORDER BY msg_count DESC`
	var users []models.User
	err := db.Select(&users, query, chat.ID)
	return users, err
}

func (db *chatsUsersService) insertRelationship(tx *sqlx.Tx, chatId int64, userId int64) error {
	const query = `INSERT INTO 
    chats_users (chat_id, user_id, in_group) 
    VALUES (?, ?, true)
    ON DUPLICATE KEY UPDATE chat_id = chat_id, msg_count = msg_count + 1, in_group = true`
	_, err := tx.Exec(query, chatId, userId)
	return err
}

func (db *chatsUsersService) IsAllowed(chat *telebot.Chat, user *telebot.User) bool {
	if utils.IsAdmin(user) {
		return true
	}

	const query = `SELECT 1 FROM chats, users 
	WHERE chats.id = ?
	AND chats.allowed = true
	OR (users.id = ? AND users.allowed = true);`

	var isAllowed bool
	err := db.Get(&isAllowed, query, chat.ID, user.ID)
	if err != nil {
		return false
	}
	return isAllowed
}

func (db *chatsUsersService) Leave(chat *telebot.Chat, user *telebot.User) error {
	const query = `UPDATE chats_users SET in_group = false
	WHERE chat_id = ?
	  AND user_id = ?`

	_, err := db.Exec(query, chat.ID, user.ID)
	return err
}

func (db *chatsUsersService) GetAllUsersInChat(chat *telebot.Chat) ([]models.User, error) {
	const query = `SELECT u.id, u.first_name, u.last_name FROM chats_users
	JOIN users u on u.id = chats_users.user_id
	WHERE chat_id = ?
	AND in_group = true
	ORDER BY u.first_name, u.last_name`
	var users []models.User
	err := db.Select(&users, query, chat.ID)
	return users, err
}
