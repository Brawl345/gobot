package sql

import (
	"errors"
	"sync"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type credentialService struct {
	*sqlx.DB
	log         *logger.Logger
	mu          sync.RWMutex
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
		log.Err(err).Msg("Failed to load credentials")
	} else {
		for _, cred := range credentials {
			s.credentials[cred.Name] = cred.Value
		}
	}

	return s
}

func (db *credentialService) GetAllCredentials() map[string]string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	credentials := make(map[string]string, len(db.credentials))
	for name, value := range db.credentials {
		credentials[name] = value
	}
	return credentials
}

func (db *credentialService) GetKey(name string) string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.credentials[name]
}

func (db *credentialService) SetKey(name, value string) error {
	const query = `INSERT INTO credentials (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?`
	_, err := db.Exec(query, name, value, value)

	if err == nil {
		db.mu.Lock()
		db.credentials[name] = value
		db.mu.Unlock()
	}

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

	db.mu.Lock()
	delete(db.credentials, name)
	db.mu.Unlock()

	return err
}
