package storage

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	UserStorage interface {
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
		Allowed   bool           `db:"allowed"`
		MsgCount  int64          `db:"msg_count"`
	}
)

func (db *Users) Allow(user *telebot.User) error {
	const query = `UPDATE users SET allowed = true WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *Users) Create(user *telebot.User) error {
	const query = `INSERT INTO 
    users (id, first_name, last_name)
    VALUES (? ,?, ?)
    ON DUPLICATE KEY UPDATE first_name = ?, last_name = ?`
	_, err := db.Exec(
		query,
		user.ID,
		user.FirstName,
		NewNullString(user.LastName),
		user.FirstName,
		NewNullString(user.LastName),
	)
	return err
}

func (db *Users) CreateTx(tx *sqlx.Tx, user *telebot.User) error {
	const query = `INSERT INTO 
    users (id, first_name, last_name)
    VALUES (? ,?, ?)
    ON DUPLICATE KEY UPDATE first_name = ?, last_name = ?`
	_, err := tx.Exec(
		query,
		user.ID,
		user.FirstName,
		NewNullString(user.LastName),
		user.FirstName,
		NewNullString(user.LastName),
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
