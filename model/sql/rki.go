package sql

import (
	"database/sql"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type rkiService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewRKIService(db *sqlx.DB) *rkiService {
	return &rkiService{
		DB:  db,
		log: logger.New("rkiService"),
	}
}

func (db *rkiService) DelAGS(user *gotgbot.User) error {
	const query = `UPDATE users SET rki_ags = NULL WHERE id = ?`
	_, err := db.Exec(query, user.Id)
	return err
}

func (db *rkiService) SetAGS(user *gotgbot.User, ags string) error {
	const query = `UPDATE users SET rki_ags = ? WHERE id = ?`
	_, err := db.Exec(query, ags, user.Id)
	return err
}

func (db *rkiService) GetAGS(user *gotgbot.User) (string, error) {
	const query = `SELECT rki_ags FROM users WHERE id = ?`
	var ags sql.NullString
	err := db.Get(&ags, query, user.Id)
	return ags.String, err
}
