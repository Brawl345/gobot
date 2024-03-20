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
	const query = `DELETE FROM google_images_queries giq
   USING google_images g
       WHERE giq.id = g.query_id
       AND giq.created_at < NOW() - INTERVAL '7 DAY'`
	_, err := db.Exec(query)
	return err
}
