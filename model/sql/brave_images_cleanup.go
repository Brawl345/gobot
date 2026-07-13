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
	const query = `DELETE biq, b FROM brave_images_queries biq
   RIGHT JOIN brave_images b ON biq.id = b.query_id
   WHERE biq.created_at < NOW() - INTERVAL 7 DAY`
	_, err := db.Exec(query)
	return err
}
