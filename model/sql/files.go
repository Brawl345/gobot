package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type fileService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewFileService(db *sqlx.DB) *fileService {
	return &fileService{
		DB:  db,
		log: logger.New("fileService"),
	}
}

func (db *fileService) Create(uniqueID, fileName, mediaType string) error {
	const query = `INSERT INTO files (id, file_name, type) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, uniqueID, fileName, mediaType)
	return err
}

func (db *fileService) Exists(uniqueID string) (bool, error) {
	const query = `SELECT 1 FROM files 
         WHERE id = $1`

	var exists bool
	err := db.Get(&exists, query, uniqueID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists, err
}
