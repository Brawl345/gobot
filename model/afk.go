package model

import (
	"database/sql"
	"time"
)

type AFKData struct {
	Since     time.Time      `db:"afk_since"`
	Reason    sql.NullString `db:"afk_reason"`
	FirstName string         `db:"first_name"`
}

func (a *AFKData) Duration() time.Duration {
	return time.Since(a.Since)
}
