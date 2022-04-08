package sql

import "github.com/jmoiron/sqlx"

type FileService struct {
	*sqlx.DB
}

func NewFileService(db *sqlx.DB) *FileService {
	return &FileService{db}
}

func (db *FileService) Create(uniqueID, fileName, mediaType string) error {
	const query = `INSERT INTO files (id, file_name, type) VALUES (?, ?, ?)`
	_, err := db.Exec(query, uniqueID, fileName, mediaType)
	return err
}

func (db *FileService) Exists(uniqueID string) (bool, error) {
	const query = `SELECT 1 FROM files
WHERE id = ?`

	var exists bool
	err := db.Get(&exists, query, uniqueID)
	return exists, err
}
