package sql

import (
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type birthdayService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewBirthdayService(db *sqlx.DB) *birthdayService {
	return &birthdayService{
		DB:  db,
		log: logger.New("birthdayService"),
	}
}

func (db *birthdayService) BirthdayNotificationsEnabled(chat *telebot.Chat) (bool, error) {
	const query = `SELECT birthday_notifications_enabled FROM chats WHERE id = ?`
	var enabled bool
	err := db.Get(&enabled, query, chat.ID)
	return enabled, err
}

func (db *birthdayService) EnableBirthdayNotifications(chat *telebot.Chat) error {
	enabled, err := db.BirthdayNotificationsEnabled(chat)
	if err != nil {
		return err
	}
	if enabled {
		return model.ErrAlreadyExists
	}

	const query = `UPDATE chats SET birthday_notifications_enabled = true WHERE id = ?`
	_, err = db.Exec(query, chat.ID)
	return err
}

func (db *birthdayService) DisableBirthdayNotifications(chat *telebot.Chat) error {
	enabled, err := db.BirthdayNotificationsEnabled(chat)
	if err != nil {
		return err
	}
	if !enabled {
		return model.ErrAlreadyExists
	}

	const query = `UPDATE chats SET birthday_notifications_enabled = false WHERE id = ?`
	_, err = db.Exec(query, chat.ID)
	return err
}

func (db *birthdayService) SetBirthday(user *telebot.User, birthday time.Time) error {
	const query = `UPDATE users SET birthday = ? WHERE id = ?`
	_, err := db.Exec(query, birthday, user.ID)
	return err
}

func (db *birthdayService) DeleteBirthday(user *telebot.User) error {
	const query = `UPDATE users SET birthday = NULL WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *birthdayService) Birthdays(chat *telebot.Chat) ([]model.User, error) {
	const query = `SELECT u.first_name, u.last_name, u.birthday FROM chats_users
	JOIN users u on u.id = chats_users.user_id
	WHERE chat_id = ?
	AND in_group = true
	AND u.birthday IS NOT NULL
	ORDER BY u.birthday`
	var users []model.User
	err := db.Select(&users, query, chat.ID)
	return users, err
}

func (db *birthdayService) TodaysBirthdays() (map[int64][]model.User, error) {
	const query = `SELECT u.first_name, u.last_name, u.birthday, cu.chat_id FROM chats_users cu
	LEFT JOIN users u ON u.id = cu.user_id
	LEFT JOIN chats c ON c.id = cu.chat_id
	WHERE c.birthday_notifications_enabled = true
  	AND cu.in_group = true
	AND DAYOFMONTH(u.birthday) = DAYOFMONTH(NOW())
	AND MONTH(u.birthday) = MONTH(NOW())`
	birthdayList := make(map[int64][]model.User)

	rows, _ := db.Queryx(query)
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	for rows.Next() {
		var chatID int64
		var user model.User
		err := rows.Scan(&user.FirstName, &user.LastName, &user.Birthday, &chatID)
		if err != nil {
			return nil, err
		}

		birthdayList[chatID] = append(birthdayList[chatID], user)
	}

	return birthdayList, nil
}
