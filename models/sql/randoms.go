package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
)

type randomService struct {
	*sqlx.DB
}

func NewRandomService(db *sqlx.DB) *randomService {
	return &randomService{db}
}

func (db *randomService) exists(random string) (bool, error) {
	const quoery = `SELECT 1 FROM randoms WHERE text = ?`
	var exists bool
	err := db.Get(&exists, quoery, random)
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
		return models.ErrNotFound
	}

	const query = `DELETE FROM randoms WHERE text = ?`

	_, err = db.Exec(query, random)
	return err
}

func (db *randomService) GetRandom() (string, error) {
	var random string
	err := db.Get(&random, "SELECT text FROM randoms ORDER BY RAND() LIMIT 1")
	if errors.Is(err, sql.ErrNoRows) {
		return "", models.ErrNotFound
	}
	return random, err
}

func (db *randomService) SaveRandom(random string) error {
	exists, err := db.exists(random)
	if err != nil {
		return err
	}
	if exists {
		return models.ErrAlreadyExists
	}

	const query = `INSERT INTO randoms (text) VALUES (?)`
	_, err = db.Exec(query, random)
	return err
}
