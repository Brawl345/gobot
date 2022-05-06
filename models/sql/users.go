package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type userService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewUserService(db *sqlx.DB) *userService {
	return &userService{
		DB:  db,
		log: logger.New("userService"),
	}
}

func (db *userService) Allow(user *telebot.User) error {
	const query = `UPDATE users SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *userService) Create(user *telebot.User) error {
	const query = `INSERT INTO 
    users (id, first_name, last_name, username)
    VALUES (? ,?, ?, ?)
    ON DUPLICATE KEY UPDATE first_name = ?, last_name = ?, username = ?`
	_, err := db.Exec(
		query,
		user.ID,
		user.FirstName,
		NewNullString(user.LastName),
		NewNullString(user.Username),
		user.FirstName,
		NewNullString(user.LastName),
		NewNullString(user.Username),
	)
	return err
}

func (db *userService) CreateTx(tx *sqlx.Tx, user *telebot.User) error {
	const query = `INSERT INTO 
    users (id, first_name, last_name, username)
    VALUES (? ,?, ?, ?)
    ON DUPLICATE KEY UPDATE first_name = ?, last_name = ?, username = ?`
	_, err := tx.Exec(
		query,
		user.ID,
		user.FirstName,
		NewNullString(user.LastName),
		NewNullString(user.Username),
		user.FirstName,
		NewNullString(user.LastName),
		NewNullString(user.Username),
	)
	return err
}

func (db *userService) Deny(user *telebot.User) error {
	const query = `UPDATE users SET allowed = false WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *userService) GetAllAllowed() ([]int64, error) {
	const query = `SELECT id FROM users WHERE allowed = true`

	var allowed []int64
	err := db.Select(&allowed, query)

	return allowed, err
}
