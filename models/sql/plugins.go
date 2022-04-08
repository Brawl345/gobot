package sql

import (
	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
)

type PluginService struct {
	*sqlx.DB
}

func NewPluginService(db *sqlx.DB) *PluginService {
	return &PluginService{db}
}

func (db *PluginService) CreateTx(tx *sqlx.Tx, pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
    ON DUPLICATE KEY UPDATE name = name`
	_, err := tx.Exec(query, pluginName)
	return err
}

func (db *PluginService) Disable(pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
	ON DUPLICATE KEY UPDATE enabled = false`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *PluginService) Enable(pluginName string) error {
	const query = `INSERT INTO plugins (name) VALUES (?) ON DUPLICATE KEY UPDATE enabled = true`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *PluginService) GetAllEnabled() ([]string, error) {
	const query = `SELECT name, enabled FROM plugins WHERE enabled = 1`

	var enabledPlugins []string
	var plugins []models.Plugin
	err := db.Select(&plugins, query)

	for _, plugin := range plugins {
		enabledPlugins = append(enabledPlugins, plugin.Name)
	}

	return enabledPlugins, err
}
