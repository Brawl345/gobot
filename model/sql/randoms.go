package sql

import (
	"database/sql"
	"errors"
	"math/rand"

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

func (db *randomService) DeleteRandom(random string) error {
	const query = `DELETE FROM randoms WHERE text = ?`

	res, err := db.Exec(query, random)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (db *randomService) GetRandom() (string, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM randoms")
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", model.ErrNotFound
	}

	offset := rand.Intn(count)
	var random string
	err = db.Get(&random, "SELECT text FROM randoms LIMIT 1 OFFSET ?", offset)
	if errors.Is(err, sql.ErrNoRows) {
		return "", model.ErrNotFound
	}
	return random, err
}

func (db *randomService) SaveRandom(random string) error {
	const query = `INSERT INTO randoms (text)
	SELECT ? FROM DUAL
	WHERE NOT EXISTS (SELECT 1 FROM randoms WHERE text = ?)`

	res, err := db.Exec(query, random, random)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrAlreadyExists
	}
	return nil
}
