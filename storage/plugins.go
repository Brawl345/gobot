package storage

import "github.com/jmoiron/sqlx"

type (
	PluginStorage interface {
		CreateTx(tx *sqlx.Tx, pluginName string) error
		Disable(pluginName string) error
		Enable(pluginName string) error
		GetAllEnabled() ([]string, error)
	}

	Plugins struct {
		*sqlx.DB
	}

	Plugin struct {
		Name    string `db:"name"`
		Enabled bool   `db:"enabled"`
	}
)

func (db *Plugins) CreateTx(tx *sqlx.Tx, pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
    ON DUPLICATE KEY UPDATE name = name`
	_, err := tx.Exec(query, pluginName)
	return err
}

func (db *Plugins) Disable(pluginName string) error {
	const query = `INSERT INTO plugins 
	(name, enabled) 
	VALUES (?, false)
	ON DUPLICATE KEY UPDATE enabled = false`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *Plugins) Enable(pluginName string) error {
	const query = `INSERT INTO plugins (name) VALUES (?) ON DUPLICATE KEY UPDATE enabled = true`
	_, err := db.Exec(query, pluginName)
	return err
}

func (db *Plugins) GetAllEnabled() ([]string, error) {
	const query = `SELECT name, enabled FROM plugins WHERE enabled = 1`

	var enabledPlugins []string
	var plugins []Plugin
	err := db.Select(&plugins, query)

	for _, plugin := range plugins {
		enabledPlugins = append(enabledPlugins, plugin.Name)
	}

	return enabledPlugins, err
}
