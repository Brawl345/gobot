package models

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

func (user *User) GetFullName() string {
	if user.LastName.Valid {
		return user.FirstName + " " + user.LastName.String
	}
	return user.FirstName
}
