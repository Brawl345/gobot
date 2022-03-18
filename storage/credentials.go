package storage

import (
	"errors"
	"github.com/jmoiron/sqlx"
)

type (
	CredentialStorage interface {
		GetAllCredentials() ([]Credential, error)
		GetKey(name string) (string, error)
		SetKey(name, value string) error
		DeleteKey(name string) error
	}

	Credentials struct {
		*sqlx.DB
	}

	Credential struct {
		Name  string `db:"name"`
		Value string `db:"value"`
	}
)

func (db *Credentials) GetAllCredentials() ([]Credential, error) {
	const query = `SELECT name, value FROM credentials ORDER BY name DESC`
	var credentials []Credential
	err := db.Select(&credentials, query)
	return credentials, err
}

func (db *Credentials) GetKey(name string) (string, error) {
	const query = `SELECT value FROM credentials WHERE name = ?`
	var value string
	err := db.Get(&value, query, name)
	return value, err
}

func (db *Credentials) SetKey(name, value string) error {
	const query = `INSERT INTO credentials (name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?`
	_, err := db.Exec(query, name, value, value)
	return err
}

func (db *Credentials) DeleteKey(name string) error {
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
