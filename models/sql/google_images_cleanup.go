package sql

import "github.com/jmoiron/sqlx"

type googleImagesCleanupService struct {
	*sqlx.DB
}

func NewGoogleImagesCleanupService(db *sqlx.DB) *googleImagesCleanupService {
	return &googleImagesCleanupService{db}
}

func (db *googleImagesCleanupService) Cleanup() error {
	const query = `DELETE giq, g FROM google_images_queries giq
   RIGHT JOIN google_images g ON giq.id = g.query_id
   WHERE giq.created_at < NOW() - INTERVAL 7 DAY`
	_, err := db.Exec(query)
	return err
}
