package sql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type homeService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewHomeService(db *sqlx.DB) *homeService {
	return &homeService{
		DB:  db,
		log: logger.New("homeService"),
	}
}

func (db *homeService) GetHome(user *gotgbot.User) (gotgbot.Venue, error) {
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
	err := db.Get(&geocoding, query, user.Id)
	if err != nil {
		return gotgbot.Venue{}, nil
	}

	if !geocoding.Address.Valid {
		return gotgbot.Venue{}, model.ErrHomeAddressNotSet
	}

	return gotgbot.Venue{
		Title:   "Festgelegter Wohnort",
		Address: geocoding.Address.String,
		Location: gotgbot.Location{
			Latitude:  geocoding.Lat.Float64,
			Longitude: geocoding.Lng.Float64,
		},
	}, nil
}

func (db *homeService) SetHome(user *gotgbot.User, venue *gotgbot.Venue) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

	const insertAddressQuery = `INSERT INTO geocoding 
    (address, latitude, longitude) 
	VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`
	res, err := db.Exec(insertAddressQuery, venue.Address, venue.Location.Latitude, venue.Location.Longitude)
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
	_, err = tx.Exec(insertHomeQuery, lastInsertId, user.Id)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return err
}

func (db *homeService) DeleteHome(user *gotgbot.User) error {
	const query = `UPDATE users
	SET HOME = NULL
	WHERE id = ?`

	_, err := db.Exec(query, user.Id)
	return err
}
