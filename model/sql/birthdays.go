package sql

import (
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
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

func (db *birthdayService) BirthdayNotificationsEnabled(chat *gotgbot.Chat) (bool, error) {
	const query = `SELECT birthday_notifications_enabled FROM chats WHERE id = $1`
	var enabled bool
	err := db.Get(&enabled, query, chat.Id)
	return enabled, err
}

func (db *birthdayService) EnableBirthdayNotifications(chat *gotgbot.Chat) error {
	enabled, err := db.BirthdayNotificationsEnabled(chat)
	if err != nil {
		return err
	}
	if enabled {
		return model.ErrAlreadyExists
	}

	const query = `UPDATE chats SET birthday_notifications_enabled = true WHERE id = $1`
	_, err = db.Exec(query, chat.Id)
	return err
}

func (db *birthdayService) DisableBirthdayNotifications(chat *gotgbot.Chat) error {
	enabled, err := db.BirthdayNotificationsEnabled(chat)
	if err != nil {
		return err
	}
	if !enabled {
		return model.ErrAlreadyExists
	}

	const query = `UPDATE chats SET birthday_notifications_enabled = false WHERE id = $1`
	_, err = db.Exec(query, chat.Id)
	return err
}

func (db *birthdayService) SetBirthday(user *gotgbot.User, birthday time.Time) error {
	const query = `UPDATE users SET birthday = $1 WHERE id = $2`
	_, err := db.Exec(query, birthday, user.Id)
	return err
}

func (db *birthdayService) DeleteBirthday(user *gotgbot.User) error {
	const query = `UPDATE users SET birthday = NULL WHERE id = $1`
	_, err := db.Exec(query, user.Id)
	return err
}

func (db *birthdayService) Birthdays(chat *gotgbot.Chat) ([]model.User, error) {
	const query = `SELECT u.first_name, u.last_name, u.birthday FROM chats_users
	JOIN users u on u.id = chats_users.user_id
	WHERE chat_id = $1
	AND in_group = true
	AND u.birthday IS NOT NULL
	ORDER BY u.birthday`
	var users []model.User
	err := db.Select(&users, query, chat.Id)
	return users, err
}

func (db *birthdayService) TodaysBirthdays() (map[int64][]model.User, error) {
	const query = `SELECT u.first_name, u.last_name, u.birthday, cu.chat_id FROM chats_users cu
	LEFT JOIN users u ON u.id = cu.user_id
	LEFT JOIN chats c ON c.id = cu.chat_id
	WHERE c.birthday_notifications_enabled = true
  	AND cu.in_group = true
	AND EXTRACT(DAY FROM u.birthday) = EXTRACT(DAY FROM CURRENT_DATE)
	AND EXTRACT(MONTH FROM u.birthday) = EXTRACT(MONTH FROM CURRENT_DATE)`
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
