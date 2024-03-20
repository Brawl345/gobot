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

type reminderService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewReminderService(db *sqlx.DB) *reminderService {
	return &reminderService{
		DB:  db,
		log: logger.New("reminderService"),
	}
}

func (db *reminderService) DeleteReminder(chat *gotgbot.Chat, user *gotgbot.User, id string) error {
	var exists bool
	var err error
	if chat.Type == gotgbot.ChatTypePrivate {
		const existsQuery = `SELECT 1 FROM reminders WHERE id = $1 AND chat_id IS NULL AND user_id = $2`
		err = db.Get(&exists, existsQuery, id, user.Id)
	} else {
		const existsQuery = `SELECT 1 FROM reminders WHERE id = $1 AND chat_id = $2`
		err = db.Get(&exists, existsQuery, id, chat.Id)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrNotFound
		}
		return err
	}

	if !exists {
		return model.ErrNotFound
	}

	if chat.Type == gotgbot.ChatTypePrivate {
		const query = `DELETE FROM reminders WHERE id = $1 AND chat_id IS NULL AND user_id = $2`
		_, err = db.Exec(query, id, user.Id)
	} else {
		const query = `DELETE FROM reminders WHERE id = $1 AND chat_id = $2`
		_, err = db.Exec(query, id, chat.Id)
	}

	return err
}

func (db *reminderService) DeleteReminderByID(id int64) error {
	const query = `DELETE FROM reminders WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}

func (db *reminderService) GetAllReminders() ([]model.Reminder, error) {
	const query = `SELECT id, time FROM reminders`
	var reminders []model.Reminder
	err := db.Select(&reminders, query)
	return reminders, err
}

func (db *reminderService) GetReminderByID(id int64) (model.Reminder, error) {
	const query = `SELECT chat_id, user_id, username, text FROM reminders r 
    RIGHT JOIN users u ON r.user_id = u.id
	WHERE r.id = $1`
	var reminder model.Reminder
	err := db.Get(&reminder, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reminder, model.ErrNotFound
		}
	}

	return reminder, err
}

func (db *reminderService) GetReminders(chat *gotgbot.Chat, user *gotgbot.User) ([]model.Reminder, error) {
	var err error
	var reminders []model.Reminder

	if chat.Type == gotgbot.ChatTypePrivate {
		const query = `SELECT id, time, text FROM reminders WHERE chat_id IS NULL AND user_id = $1 ORDER BY time`
		err = db.Select(&reminders, query, user.Id)
	} else {
		const query = `SELECT id, time, text FROM reminders WHERE chat_id = $1 ORDER BY time`
		err = db.Select(&reminders, query, chat.Id)
	}

	return reminders, err
}

func (db *reminderService) SaveReminder(
	chat *gotgbot.Chat,
	user *gotgbot.User,
	remindAt time.Time,
	text string,
) (int64, error) {
	var err error
	var lastInsertID int64
	if chat.Type == gotgbot.ChatTypePrivate {
		const query = `INSERT INTO reminders (user_id, time, text) VALUES ($1, $2, $3) RETURNING id`
		err = db.QueryRow(query, user.Id, remindAt, text).Scan(&lastInsertID)
	} else {
		const query = `INSERT INTO reminders (chat_id, user_id, time, text) VALUES ($1, $2, $3, $4) RETURNING id`
		err = db.QueryRow(query, chat.Id, user.Id, remindAt, text).Scan(&lastInsertID)
	}
	if err != nil {
		return 0, err
	}
	return lastInsertID, err
}
