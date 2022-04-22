package sql

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type homeService struct {
	*sqlx.DB
}

func NewHomeService(db *sqlx.DB) *homeService {
	return &homeService{db}
}

func (db *homeService) GetHome(user *telebot.User) (*telebot.Venue, error) {
	const query = `SELECT address, latitude, longitude
	FROM geocoding g
	RIGHT OUTER JOIN users u ON u.home = g.id
	WHERE u.id = ?`

	type Home struct {
		Address sql.NullString  `db:"address"`
		Lat     sql.NullFloat64 `db:"latitude"`
		Lng     sql.NullFloat64 `db:"longitude"`
	}

	var geocoding Home
	err := db.Get(&geocoding, query, user.ID)
	if err != nil {
		return nil, nil
	}

	if !geocoding.Address.Valid {
		return nil, errors.New("no home address set")
	}

	return &telebot.Venue{
		Title:   "Festgelegter Wohnort",
		Address: geocoding.Address.String,
		Location: telebot.Location{
			Lat: float32(geocoding.Lat.Float64),
			Lng: float32(geocoding.Lng.Float64),
		},
	}, nil
}

func (db *homeService) SetHome(user *telebot.User, venue *telebot.Venue) error {
	const query = `UPDATE users
	SET HOME = (SELECT id FROM geocoding g
	WHERE g.address = ?)
	WHERE id = ?`

	_, err := db.Exec(query, venue.Address, user.ID)
	return err
}

func (db *homeService) DeleteHome(user *telebot.User) error {
	const query = `UPDATE users
	SET HOME = NULL
	WHERE id = ?`

	_, err := db.Exec(query, user.ID)
	return err
}
