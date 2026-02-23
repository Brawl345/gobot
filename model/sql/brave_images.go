package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type (
	braveImagesService struct {
		*sqlx.DB
		log *logger.Logger
	}
)

func NewBraveImagesService(db *sqlx.DB) *braveImagesService {
	return &braveImagesService{
		DB:  db,
		log: logger.New("braveImagesService"),
	}
}

func (db *braveImagesService) GetImages(query string) (model.ImageSearchImages, error) {
	query = strings.ToLower(query)
	const selectQuery = `SELECT query_id, image_url, context_url, is_gif, current_index
		FROM brave_images bi
		RIGHT JOIN brave_images_queries biq ON biq.id = bi.query_id
		WHERE query = ?`

	rows, err := db.Queryx(selectQuery, query)
	if err != nil {
		return model.ImageSearchImages{}, err
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	var images []model.ImageSearchImage
	var currentIndex int
	var queryID int64
	for rows.Next() {
		var image Image
		err := rows.Scan(&queryID, &image.ImageURL, &image.ContextURL, &image.GIF, &currentIndex)
		if err != nil {
			db.log.Err(err).Send()
			return model.ImageSearchImages{}, err
		}
		images = append(images, image)
	}

	return model.ImageSearchImages{
		CurrentIndex: currentIndex,
		QueryID:      queryID,
		Images:       images,
	}, nil
}

func (db *braveImagesService) GetImagesFromQueryID(queryID int64) (model.ImageSearchImages, error) {
	const selectQuery = `SELECT image_url, context_url, is_gif, current_index
		FROM brave_images bi
		RIGHT JOIN brave_images_queries biq ON biq.id = bi.query_id
		WHERE query_id = ?`

	rows, err := db.Queryx(selectQuery, queryID)
	if err != nil {
		return model.ImageSearchImages{}, err
	}
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	var images []model.ImageSearchImage
	var currentIndex int
	for rows.Next() {
		var image Image
		err := rows.Scan(&image.ImageURL, &image.ContextURL, &image.GIF, &currentIndex)
		if err != nil {
			db.log.Err(err).Send()
			return model.ImageSearchImages{}, err
		}
		images = append(images, image)
	}

	return model.ImageSearchImages{
		CurrentIndex: currentIndex,
		QueryID:      queryID,
		Images:       images,
	}, nil

}

func (db *braveImagesService) SaveImages(query string, wrapper *model.ImageSearchImages) (int64, error) {
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

	const insertSearchQuery = `INSERT INTO brave_images_queries (query, current_index) VALUES (?, ?)`
	res, err := tx.Exec(insertSearchQuery, query, wrapper.CurrentIndex)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if len(wrapper.Images) > 0 {
		valueStrings := make([]string, 0, len(wrapper.Images))
		valueArgs := make([]interface{}, 0, len(wrapper.Images)*4)
		for _, image := range wrapper.Images {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			valueArgs = append(valueArgs, lastInsertID, image.ImageLink(), image.ContextLink(), image.IsGIF())
		}
		insertImages := fmt.Sprintf("INSERT INTO brave_images (query_id, image_url, context_url, is_gif) VALUES %s",
			strings.Join(valueStrings, ","))
		_, err = tx.Exec(insertImages, valueArgs...)
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return lastInsertID, nil
}

func (db *braveImagesService) SaveIndex(queryID int64, index int) error {
	const updateQuery = `UPDATE brave_images_queries SET current_index = ? WHERE id = ?`
	_, err := db.Exec(updateQuery, index, queryID)
	return err
}
