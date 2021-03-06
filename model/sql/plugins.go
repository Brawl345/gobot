package sql

import (
	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type pluginService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewPluginService(db *sqlx.DB) *pluginService {
	return &pluginService{
		DB:  db,
		log: logger.New("pluginService"),
	}
}

func (db *pluginService) CreateTx(tx *sqlx.Tx, pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
    ON DUPLICATE KEY UPDATE name = name`
	_, err := tx.Exec(query, pluginName)
	return err
}

func (db *pluginService) Disable(pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
	ON DUPLICATE KEY UPDATE enabled = false`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *pluginService) Enable(pluginName string) error {
	const query = `INSERT INTO plugins (name) VALUES (?) ON DUPLICATE KEY UPDATE enabled = true`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *pluginService) GetAllEnabled() ([]string, error) {
	const query = `SELECT name, enabled FROM plugins WHERE enabled = 1`

	var enabledPlugins []string
	var plugins []model.Plugin
	err := db.Select(&plugins, query)

	for _, plugin := range plugins {
		enabledPlugins = append(enabledPlugins, plugin.Name)
	}

	return enabledPlugins, err
}
