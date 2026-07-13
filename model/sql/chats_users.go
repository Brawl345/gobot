package sql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type chatsUsersService struct {
	Chats model.ChatService
	Users model.UserService
	*sqlx.DB
	log *logger.Logger
}

func NewChatsUsersService(db *sqlx.DB, chatService model.ChatService, userService model.UserService) *chatsUsersService {
	return &chatsUsersService{
		Chats: chatService,
		Users: userService,
		DB:    db,
		log:   logger.New("chatsUsersService"),
	}
}

func (db *chatsUsersService) Create(chat *gotgbot.Chat, user *gotgbot.User) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	err = db.Users.CreateTx(tx, user)
	if err != nil {
		return err
	}

	err = db.insertRelationship(tx, chat.Id, user.Id)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *chatsUsersService) CreateBatch(chat *gotgbot.Chat, users *[]gotgbot.User) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	userValues := make([]string, 0, len(*users))
	userArgs := make([]any, 0, len(*users)*4)
	relValues := make([]string, 0, len(*users))
	relArgs := make([]any, 0, len(*users)*2)

	for _, user := range *users {
		if user.IsBot {
			continue
		}

		userValues = append(userValues, "(?, ?, ?, ?)")
		userArgs = append(userArgs, user.Id, user.FirstName, NewNullString(user.LastName), NewNullString(user.Username))

		relValues = append(relValues, "(?, ?, 0, true)")
		relArgs = append(relArgs, chat.Id, user.Id)
	}

	if len(userValues) > 0 {
		userQuery := `INSERT INTO users (id, first_name, last_name, username) VALUES ` +
			strings.Join(userValues, ", ") +
			` ON DUPLICATE KEY UPDATE first_name = VALUES(first_name), last_name = VALUES(last_name), username = VALUES(username)`
		if _, err := tx.Exec(userQuery, userArgs...); err != nil {
			return err
		}

		relQuery := `INSERT INTO chats_users (chat_id, user_id, msg_count, in_group) VALUES ` +
			strings.Join(relValues, ", ") +
			` ON DUPLICATE KEY UPDATE in_group = true`
		if _, err := tx.Exec(relQuery, relArgs...); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *chatsUsersService) GetAllUsersWithMsgCount(chat *gotgbot.Chat) ([]model.User, error) {
	const query = `SELECT u.first_name, u.last_name, msg_count, in_group FROM chats_users
JOIN users u on u.id = chats_users.user_id
WHERE chat_id = ?
ORDER BY msg_count DESC`
	var users []model.User
	err := db.Select(&users, query, chat.Id)
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

func (db *chatsUsersService) IsAllowed(chat *gotgbot.Chat, user *gotgbot.User) bool {
	if tgUtils.IsAdmin(user) {
		return true
	}

	const query = `SELECT
		EXISTS(SELECT 1 FROM chats WHERE id = ? AND allowed = true)
		OR EXISTS(SELECT 1 FROM users WHERE id = ? AND allowed = true)`

	var isAllowed bool
	err := db.Get(&isAllowed, query, chat.Id, user.Id)
	if err != nil {
		return false
	}
	return isAllowed
}

func (db *chatsUsersService) Leave(chat *gotgbot.Chat, user *gotgbot.User) error {
	const query = `UPDATE chats_users SET in_group = false
	WHERE chat_id = ?
	  AND user_id = ?`

	_, err := db.Exec(query, chat.Id, user.Id)
	return err
}

func (db *chatsUsersService) GetAllUsersInChat(chat *gotgbot.Chat) ([]model.User, error) {
	const query = `SELECT u.id, u.first_name, u.last_name FROM chats_users
	JOIN users u on u.id = chats_users.user_id
	WHERE chat_id = ?
	AND in_group = true
	ORDER BY u.first_name, u.last_name`
	var users []model.User
	err := db.Select(&users, query, chat.Id)
	return users, err
}
