package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type googleImagesCleanupService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewGoogleImagesCleanupService(db *sqlx.DB) *googleImagesCleanupService {
	return &googleImagesCleanupService{
		DB:  db,
		log: logger.New("googleImagesCleanupService"),
	}
}

func (db *googleImagesCleanupService) Cleanup() error {
	const query = `DELETE giq, g FROM google_images_queries giq
   RIGHT JOIN google_images g ON giq.id = g.query_id
   WHERE giq.created_at < NOW() - INTERVAL 7 DAY`
	_, err := db.Exec(query)
	return err
}
