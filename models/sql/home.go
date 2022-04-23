package sql

import (
	"context"
	"database/sql"

	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type homeService struct {
	*sqlx.DB
}

func NewHomeService(db *sqlx.DB) *homeService {
	return &homeService{db}
}

func (db *homeService) GetHome(user *telebot.User) (telebot.Venue, error) {
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
		return telebot.Venue{}, nil
	}

	if !geocoding.Address.Valid {
		return telebot.Venue{}, models.ErrHomeAddressNotSet
	}

	return telebot.Venue{
		Title:   "Festgelegter Wohnort",
		Address: geocoding.Address.String,
		Location: telebot.Location{
			Lat: float32(geocoding.Lat.Float64),
			Lng: float32(geocoding.Lng.Float64),
		},
	}, nil
}

func (db *homeService) SetHome(user *telebot.User, venue *telebot.Venue) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	const insertAddressQuery = `INSERT INTO geocoding 
    (address, latitude, longitude) 
	VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`
	res, err := db.Exec(insertAddressQuery, venue.Address, venue.Location.Lat, venue.Location.Lng)
	if err != nil {
		return err
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return err
	}

	const insertHomeQuery = `UPDATE users
	SET home = ?
	WHERE id = ?`
	_, err = tx.Exec(insertHomeQuery, lastInsertId, user.ID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return err
}

func (db *homeService) DeleteHome(user *telebot.User) error {
	const query = `UPDATE users
	SET HOME = NULL
	WHERE id = ?`

	_, err := db.Exec(query, user.ID)
	return err
}
