package sql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
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

func (db *reminderService) DeleteReminder(chat *telebot.Chat, user *telebot.User, id string) error {
	var exists bool
	var err error
	if chat.Type == telebot.ChatPrivate {
		const existsQuery = `SELECT 1 FROM reminders WHERE id = ? AND chat_id IS NULL AND user_id = ?`
		err = db.Get(&exists, existsQuery, id, user.ID)
	} else {
		const existsQuery = `SELECT 1 FROM reminders WHERE id = ? AND chat_id = ?`
		err = db.Get(&exists, existsQuery, id, chat.ID)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrNotFound
		}
		return err
	}

	if !exists {
		return models.ErrNotFound
	}

	if chat.Type == telebot.ChatPrivate {
		const query = `DELETE FROM reminders WHERE id = ? AND chat_id IS NULL AND user_id = ?`
		_, err = db.Exec(query, id, user.ID)
	} else {
		const query = `DELETE FROM reminders WHERE id = ? AND chat_id = ?`
		_, err = db.Exec(query, id, chat.ID)
	}

	return err
}

func (db *reminderService) DeleteReminderByID(id int64) error {
	const query = `DELETE FROM reminders WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func (db *reminderService) GetAllReminders() ([]models.Reminder, error) {
	const query = `SELECT id, time FROM reminders`
	var reminders []models.Reminder
	err := db.Select(&reminders, query)
	return reminders, err
}

func (db *reminderService) GetReminderByID(id int64) (models.Reminder, error) {
	const query = `SELECT chat_id, user_id, username, text FROM reminders r 
    RIGHT JOIN users u ON r.user_id = u.id
	WHERE r.id = ?`
	var reminder models.Reminder
	err := db.Get(&reminder, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return reminder, models.ErrNotFound
		}
	}

	return reminder, err
}

func (db *reminderService) GetReminders(chat *telebot.Chat, user *telebot.User) ([]models.Reminder, error) {
	var err error
	var reminders []models.Reminder

	if chat.Type == telebot.ChatPrivate {
		const query = `SELECT id, time, text FROM reminders WHERE chat_id IS NULL AND user_id = ? ORDER BY time`
		err = db.Select(&reminders, query, user.ID)
	} else {
		const query = `SELECT id, time, text FROM reminders WHERE chat_id = ? ORDER BY time`
		err = db.Select(&reminders, query, chat.ID)
	}

	return reminders, err
}

func (db *reminderService) SaveReminder(
	chat *telebot.Chat,
	user *telebot.User,
	remindAt time.Time,
	text string,
) (int64, error) {
	var err error
	var res sql.Result
	if chat.Type == telebot.ChatPrivate {
		const query = `INSERT INTO reminders (user_id, time, text) VALUES (?, ?, ?)`
		res, err = db.Exec(query, user.ID, remindAt, text)
	} else {
		const query = `INSERT INTO reminders (chat_id, user_id, time, text) VALUES (?, ?, ?, ?)`
		res, err = db.Exec(query, chat.ID, user.ID, remindAt, text)
	}
	if err != nil {
		return 0, err
	}
	lastInsertedID, err := res.LastInsertId()
	return lastInsertedID, err
}
