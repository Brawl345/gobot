package model

import (
	"database/sql"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/jmoiron/sqlx"
)

type (
	UserService interface {
		Allow(user *gotgbot.User) error
		Create(user *gotgbot.User) error
		CreateTx(tx *sqlx.Tx, user *gotgbot.User) error
		Deny(user *gotgbot.User) error
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
		Birthday  sql.NullTime   `db:"birthday"`
	}
)

func (user *User) GetFullName() string {
	if user.LastName.Valid {
		return user.FirstName + " " + user.LastName.String
	}
	return user.FirstName
}
