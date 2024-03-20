package sql

import (
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type credentialService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewCredentialService(db *sqlx.DB) *credentialService {
	return &credentialService{
		DB:  db,
		log: logger.New("credentialService"),
	}
}

func (db *credentialService) GetAllCredentials() ([]model.Credential, error) {
	const query = `SELECT name, value FROM credentials ORDER BY name`
	var credentials []model.Credential
	err := db.Select(&credentials, query)
	return credentials, err
}

func (db *credentialService) GetKey(name string) (string, error) {
	const query = `SELECT value FROM credentials WHERE name = $1`
	var value string
	err := db.Get(&value, query, name)
	return value, err
}

func (db *credentialService) SetKey(name, value string) error {
	const query = `INSERT INTO credentials (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`
	_, err := db.Exec(query, name, value)
	return err
}

func (db *credentialService) DeleteKey(name string) error {
	const query = `DELETE FROM credentials WHERE name = $1`
	res, err := db.Exec(query, name)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if rows == 0 {
		return errors.New("❌ Key nicht gefunden")
	}
	return err
}
