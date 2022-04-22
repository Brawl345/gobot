package sql

import (
	"context"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/telebot.v3"
)

type chatsPluginsService struct {
	Chats   models.ChatService
	Plugins models.PluginService
	*sqlx.DB
}

var log = logger.New("sql")

func NewChatsPluginsService(
	db *sqlx.DB,
	chatService models.ChatService,
	pluginService models.PluginService,
) *chatsPluginsService {
	return &chatsPluginsService{
		Chats:   chatService,
		Plugins: pluginService,
		DB:      db,
	}
}

func (db *chatsPluginsService) Disable(chat *telebot.Chat, pluginName string) error {
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

func (db *chatsPluginsService) Enable(chat *telebot.Chat, pluginName string) error {
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

func (db *chatsPluginsService) insertRelationship(tx *sqlx.Tx, chat *telebot.Chat, pluginName string, enabled bool) error {
	const query = `INSERT INTO 
    chats_plugins (chat_id, plugin_name, enabled) 
    VALUES (?, ?, ?)
    ON DUPLICATE KEY UPDATE enabled = ?`
	_, err := tx.Exec(query, chat.ID, pluginName, enabled, enabled)
	return err
}

func (db *chatsPluginsService) GetAllDisabled() (map[int64][]string, error) {
	const query = `SELECT chat_id, plugin_name FROM chats_plugins WHERE enabled = false`

	rows, _ := db.Queryx(query)
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			log.Err(err).Send()
		}
	}(rows)

	disabledPlugins := make(map[int64][]string)

	for rows.Next() {
		var chatID int64
		var pluginName string
		err := rows.Scan(&chatID, &pluginName)
		if err != nil {
			log.Err(err).Send()
			return nil, err
		}

		disabledPlugins[chatID] = append(disabledPlugins[chatID], pluginName)
	}

	return disabledPlugins, nil
}
