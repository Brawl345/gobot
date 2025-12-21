package sql

import (
	"database/sql"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type gelbooruService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewGelbooruService(db *sqlx.DB) *gelbooruService {
	return &gelbooruService{
		DB:  db,
		log: logger.New("gelbooruService"),
	}
}

func (db *gelbooruService) GetQuery(queryID int64) (string, error) {
	const selectQuery = `SELECT query FROM gelbooru_queries WHERE id = ?`
	var query string
	err := db.Get(&query, selectQuery, queryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", model.ErrQueryNotFound
		}
		return "", err
	}
	return query, nil
}

func (db *gelbooruService) SaveQuery(query string) (int64, error) {
	query = strings.ToLower(query)

	const selectQuery = `SELECT id FROM gelbooru_queries WHERE query = ?`
	var existingID int64
	err := db.Get(&existingID, selectQuery, query)
	if err == nil {
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	const insertQuery = `INSERT INTO gelbooru_queries (query) VALUES (?)`
	res, err := db.Exec(insertQuery, query)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastInsertID, nil
}
