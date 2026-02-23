package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type braveImagesCleanupService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewBraveImagesCleanupService(db *sqlx.DB) *braveImagesCleanupService {
	return &braveImagesCleanupService{
		DB:  db,
		log: logger.New("braveImagesCleanupService"),
	}
}

func (db *braveImagesCleanupService) Cleanup() error {
	const query = `DELETE giq, b FROM google_images_queries giq
   RIGHT JOIN brave_images b ON giq.id = b.query_id
   WHERE giq.created_at < NOW() - INTERVAL 7 DAY`
	_, err := db.Exec(query)
	return err
}
