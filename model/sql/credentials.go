package sql

import (
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type credentialService struct {
	*sqlx.DB
	log         *logger.Logger
	credentials map[string]string
}

func NewCredentialService(db *sqlx.DB) *credentialService {
	s := &credentialService{
		DB:  db,
		log: logger.New("credentialService"),
	}

	const query = `SELECT name, value FROM credentials`
	var credentials []model.Credential
	err := db.Select(&credentials, query)

	s.credentials = make(map[string]string)

	if err != nil {
		log.Err(err)
	} else {
		for _, cred := range credentials {
			s.credentials[cred.Name] = cred.Value
		}
	}

	return s
}

func (db *credentialService) GetAllCredentials() map[string]string {
	return db.credentials
}

func (db *credentialService) GetKey(name string) string {
	return db.credentials[name]
}

func (db *credentialService) SetKey(name, value string) error {
	const query = `INSERT INTO credentials (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`
	_, err := db.Exec(query, name, value)

	if err == nil {
		db.credentials[name] = value
	}
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

	delete(db.credentials, name)

	return err
}
