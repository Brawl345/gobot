package model

import "database/sql"

type GeminiData struct {
	History   sql.NullString `db:"gemini_history"`
	ExpiresOn sql.NullTime   `db:"gemini_history_expires_on"`
}
