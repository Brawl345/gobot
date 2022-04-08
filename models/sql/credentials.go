package sql

import (
	"errors"

	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
)

type credentialService struct {
	*sqlx.DB
}

func NewCredentialService(db *sqlx.DB) *credentialService {
	return &credentialService{db}
}

func (db *credentialService) GetAllCredentials() ([]models.Credential, error) {
	const query = `SELECT name, value FROM credentials ORDER BY name DESC`
	var credentials []models.Credential
	err := db.Select(&credentials, query)
	return credentials, err
}

func (db *credentialService) GetKey(name string) (string, error) {
	const query = `SELECT value FROM credentials WHERE name = ?`
	var value string
	err := db.Get(&value, query, name)
	return value, err
}

func (db *credentialService) SetKey(name, value string) error {
	const query = `INSERT INTO credentials (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?`
	_, err := db.Exec(query, name, value, value)
	return err
}

func (db *credentialService) DeleteKey(name string) error {
	const query = `DELETE FROM credentials WHERE name = ?`
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
