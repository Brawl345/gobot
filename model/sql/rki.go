package sql

import (
	"database/sql"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
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

func (db *rkiService) DelAGS(user *telebot.User) error {
	const query = `UPDATE users SET rki_ags = NULL WHERE id = ?`
	_, err := db.Exec(query, user.ID)
	return err
}

func (db *rkiService) SetAGS(user *telebot.User, ags string) error {
	const query = `UPDATE users SET rki_ags = ? WHERE id = ?`
	_, err := db.Exec(query, ags, user.ID)
	return err
}

func (db *rkiService) GetAGS(user *telebot.User) (string, error) {
	const query = `SELECT rki_ags FROM users WHERE id = ?`
	var ags sql.NullString
	err := db.Get(&ags, query, user.ID)
	return ags.String, err
}
