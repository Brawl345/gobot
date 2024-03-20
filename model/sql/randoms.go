package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type randomService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewRandomService(db *sqlx.DB) *randomService {
	return &randomService{
		DB:  db,
		log: logger.New("randomService"),
	}
}

func (db *randomService) exists(random string) (bool, error) {
	const query = `SELECT 1 FROM randoms WHERE text = $1`
	var exists bool
	err := db.Get(&exists, query, random)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

func (db *randomService) DeleteRandom(random string) error {
	exists, err := db.exists(random)
	if err != nil {
		return err
	}
	if !exists {
		return model.ErrNotFound
	}

	const query = `DELETE FROM randoms WHERE text = $1`

	_, err = db.Exec(query, random)
	return err
}

func (db *randomService) GetRandom() (string, error) {
	var random string
	err := db.Get(&random, "SELECT text FROM randoms ORDER BY RANDOM() LIMIT 1")
	if errors.Is(err, sql.ErrNoRows) {
		return "", model.ErrNotFound
	}
	return random, err
}

func (db *randomService) SaveRandom(random string) error {
	exists, err := db.exists(random)
	if err != nil {
		return err
	}
	if exists {
		return model.ErrAlreadyExists
	}

	const query = `INSERT INTO randoms (text) VALUES ($1)`
	_, err = db.Exec(query, random)
	return err
}
