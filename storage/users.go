package storage

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	UserStorage interface {
		Create(user *telebot.User) error
		CreateWithTx(tx *sqlx.Tx, user *telebot.User) error
		IsAllowed(user *telebot.User) (bool, error)
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

func (db *Users) CreateWithTx(tx *sqlx.Tx, user *telebot.User) error {
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

func (db *Users) IsAllowed(user *telebot.User) (bool, error) {
	if isAdmin(user) {
		return true, nil
	}

	const query = `SELECT users.allowed FROM users WHERE users.id = ?`

	var isAllowed bool
	err := db.Get(&isAllowed, query, user.ID)
	return isAllowed, err
}
