package storage

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	UserService interface {
		Allow(user *telebot.User) error
		Create(user *telebot.User) error
		CreateTx(tx *sqlx.Tx, user *telebot.User) error
		Deny(user *telebot.User) error
		GetAllAllowed() ([]int64, error)
	}

	Users struct {
		*sqlx.DB
	}

	User struct {
		ID        int64          `db:"id"`
		FirstName string         `db:"first_name"`
		LastName  sql.NullString `db:"last_name"`
		Username  sql.NullString `db:"username"`
		Allowed   bool           `db:"allowed"`
		MsgCount  int64          `db:"msg_count"`
		InGroup   bool           `db:"in_group"`
	}
)

func NewUserService(db *sqlx.DB) *Users {
	return &Users{db}
}

func (user *User) GetFullName() string {
	if user.LastName.Valid {
		return user.FirstName + " " + user.LastName.String
	}
	return user.FirstName
}

func (db *Users) Allow(user *telebot.User) error {
	const query = `UPDATE users SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *Users) Create(user *telebot.User) error {
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

func (db *Users) CreateTx(tx *sqlx.Tx, user *telebot.User) error {
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

func (db *Users) Deny(user *telebot.User) error {
	const query = `UPDATE users SET allowed = false WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *Users) GetAllAllowed() ([]int64, error) {
	const query = `SELECT id FROM users WHERE allowed = true`

	var allowed []int64
	err := db.Select(&allowed, query)

	return allowed, err
}
