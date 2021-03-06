package sql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type (
	googleImagesService struct {
		*sqlx.DB
		log *logger.Logger
	}

	Image struct {
		QueryID    int64  `db:"query_id"`
		ImageURL   string `db:"image_url"`
		ContextURL string `db:"context_url"`
		GIF        bool   `db:"is_gif"`
	}
)

func (i Image) ImageLink() string {
	return i.ImageURL
}

func (i Image) ContextLink() string {
	return i.ContextURL
}

func (i Image) IsGIF() bool {
	return i.GIF
}

func NewGoogleImagesService(db *sqlx.DB) *googleImagesService {
	return &googleImagesService{
		DB:  db,
		log: logger.New("googleImagesService"),
	}
}

func (db *googleImagesService) GetImages(query string) (model.GoogleImages, error) {
	query = strings.ToLower(query)
	const selectQuery = `SELECT query_id, image_url, context_url, is_gif, current_index
		FROM google_images gi
		RIGHT JOIN google_images_queries giq ON giq.id = gi.query_id
		WHERE query = ?`

	rows, err := db.Queryx(selectQuery, query)
	if err != nil {
		return model.GoogleImages{}, err
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	var images []model.Image
	var currentIndex int
	var queryID int64
	for rows.Next() {
		var image Image
		err := rows.Scan(&queryID, &image.ImageURL, &image.ContextURL, &image.GIF, &currentIndex)
		if err != nil {
			db.log.Err(err).Send()
			return model.GoogleImages{}, err
		}
		images = append(images, image)
	}

	return model.GoogleImages{
		CurrentIndex: currentIndex,
		QueryID:      queryID,
		Images:       images,
	}, nil
}

func (db *googleImagesService) GetImagesFromQueryID(queryID int64) (model.GoogleImages, error) {
	const selectQuery = `SELECT image_url, context_url, is_gif, current_index
		FROM google_images gi
		RIGHT JOIN google_images_queries giq ON giq.id = gi.query_id
		WHERE query_id = ?`

	rows, err := db.Queryx(selectQuery, queryID)
	if err != nil {
		return model.GoogleImages{}, err
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	var images []model.Image
	var currentIndex int
	for rows.Next() {
		var image Image
		err := rows.Scan(&image.ImageURL, &image.ContextURL, &image.GIF, &currentIndex)
		if err != nil {
			db.log.Err(err).Send()
			return model.GoogleImages{}, err
		}
		images = append(images, image)
	}

	return model.GoogleImages{
		CurrentIndex: currentIndex,
		QueryID:      queryID,
		Images:       images,
	}, nil

}

func (db *googleImagesService) SaveImages(query string, wrapper *model.GoogleImages) (int64, error) {
	query = strings.ToLower(query)
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

	const insertSearchQuery = `INSERT INTO google_images_queries (query, current_index) VALUES (?, ?)`
	res, err := tx.Exec(insertSearchQuery, query, wrapper.CurrentIndex)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	const insertImages = `INSERT INTO google_images (query_id, image_url, context_url, is_gif) VALUES (?, ?, ?, ?)`
	// creating a query for every image is inefficient,
	// but idc
	for _, image := range wrapper.Images {
		_, err := tx.Exec(insertImages, lastInsertID, image.ImageLink(), image.ContextLink(), image.IsGIF())
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return lastInsertID, nil
}

func (db *googleImagesService) SaveIndex(queryID int64, index int) error {
	const updateQuery = `UPDATE google_images_queries SET current_index = ? WHERE id = ?`
	_, err := db.Exec(updateQuery, index, queryID)
	return err
}
