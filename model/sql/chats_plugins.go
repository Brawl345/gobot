package sql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/jmoiron/sqlx"
)

type chatsPluginsService struct {
	Chats   model.ChatService
	Plugins model.PluginService
	*sqlx.DB
	log *logger.Logger
}

func NewChatsPluginsService(
	db *sqlx.DB,
	chatService model.ChatService,
	pluginService model.PluginService,
) *chatsPluginsService {
	return &chatsPluginsService{
		Chats:   chatService,
		Plugins: pluginService,
		DB:      db,
		log:     logger.New("chatsPluginsService"),
	}
}

func (db *chatsPluginsService) Disable(chat *gotgbot.Chat, pluginName string) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

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

func (db *chatsPluginsService) Enable(chat *gotgbot.Chat, pluginName string) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Err(err).Msg("failed to rollback transaction")
		}
	}(tx)

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

func (db *chatsPluginsService) insertRelationship(tx *sqlx.Tx, chat *gotgbot.Chat, pluginName string, enabled bool) error {
	const query = `INSERT INTO 
    chats_plugins (chat_id, plugin_name, enabled) 
    VALUES ($1, $2, $3)
    ON CONFLICT (chat_id, plugin_name) DO UPDATE SET enabled = EXCLUDED.enabled`
	_, err := tx.Exec(query, chat.Id, pluginName, enabled)
	return err
}

func (db *chatsPluginsService) GetAllDisabled() (map[int64][]string, error) {
	const query = `SELECT chat_id, plugin_name FROM chats_plugins WHERE enabled = false`

	rows, _ := db.Queryx(query)
	defer func(rows *sqlx.Rows) {
		err := rows.Close()
		if err != nil {
			db.log.Err(err).Send()
		}
	}(rows)

	disabledPlugins := make(map[int64][]string)

	for rows.Next() {
		var chatID int64
		var pluginName string
		err := rows.Scan(&chatID, &pluginName)
		if err != nil {
			db.log.Err(err).Send()
			return nil, err
		}

		disabledPlugins[chatID] = append(disabledPlugins[chatID], pluginName)
	}

	return disabledPlugins, nil
}
