package model

import "database/sql"

type GPTData struct {
	ResponseID sql.NullString `db:"gpt_response_id"`
	ExpiresOn  sql.NullTime   `db:"gpt_response_id_expires_on"`
}
