package storage

import (
	"context"

	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type (
	ChatPluginStorage interface {
		Disable(chat *telebot.Chat, pluginName string) error
		Enable(chat *telebot.Chat, pluginName string) error
		GetAllDisabled() (map[int64][]string, error)
	}

	ChatsPlugins struct {
		Chats   ChatStorage
		Plugins PluginStorage
		*sqlx.DB
	}
)

func (db *ChatsPlugins) Disable(chat *telebot.Chat, pluginName string) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	err = db.Plugins.CreateTx(tx, pluginName)
	if err != nil {
		return err
	}

	err = db.insertRelationship(tx, chat, pluginName, false)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *ChatsPlugins) Enable(chat *telebot.Chat, pluginName string) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	err = db.Chats.CreateTx(tx, chat)
	if err != nil {
		return err
	}

	err = db.Plugins.CreateTx(tx, pluginName)
	if err != nil {
		return err
	}

	err = db.insertRelationship(tx, chat, pluginName, true)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *ChatsPlugins) insertRelationship(tx *sqlx.Tx, chat *telebot.Chat, pluginName string, enabled bool) error {
	const query = `INSERT INTO 
    chats_plugins (chat_id, plugin_name, enabled) 
    VALUES (?, ?, ?)
    ON DUPLICATE KEY UPDATE enabled = ?`
	_, err := tx.Exec(query, chat.ID, pluginName, enabled, enabled)
	return err
}

func (db *ChatsPlugins) GetAllDisabled() (map[int64][]string, error) {
	const query = `SELECT chat_id, plugin_name FROM chats_plugins WHERE enabled = false`

	rows, _ := db.Queryx(query)
	defer rows.Close()

	disabledPlugins := make(map[int64][]string)

	for rows.Next() {
		var chatID int64
		var pluginName string
		rows.Scan(&chatID, &pluginName)

		disabledPlugins[chatID] = append(disabledPlugins[chatID], pluginName)
	}

	return disabledPlugins, nil
}
