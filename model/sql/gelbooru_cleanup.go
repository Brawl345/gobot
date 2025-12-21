package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/jmoiron/sqlx"
)

type gelbooruCleanupService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewGelbooruCleanupService(db *sqlx.DB) *gelbooruCleanupService {
	return &gelbooruCleanupService{
		DB:  db,
		log: logger.New("gelbooruCleanupService"),
	}
}

func (db *gelbooruCleanupService) Cleanup() error {
	const query = `DELETE FROM gelbooru_queries WHERE created_at < NOW() - INTERVAL 7 DAY`
	_, err := db.Exec(query)
	return err
}
