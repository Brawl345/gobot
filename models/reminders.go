package models

import (
	"database/sql"
	"time"
)

type Reminder struct {
	ID       int64         `db:"id"`
	ChatID   sql.NullInt64 `db:"chat_id"`
	UserID   int64         `db:"user_id"`
	Username string        `db:"username"`
	Time     time.Time     `db:"time"`
	Text     string        `db:"text"`
}
