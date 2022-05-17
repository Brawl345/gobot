package model

import "github.com/jmoiron/sqlx"

type (
	PluginService interface {
		CreateTx(tx *sqlx.Tx, pluginName string) error
		Disable(pluginName string) error
		Enable(pluginName string) error
		GetAllEnabled() ([]string, error)
	}

	Plugin struct {
		Name    string `db:"name"`
		Enabled bool   `db:"enabled"`
	}
)
