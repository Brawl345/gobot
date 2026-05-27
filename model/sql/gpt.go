package sql

import (
	"database/sql"
	"errors"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/jmoiron/sqlx"
)

type gptService struct {
	*sqlx.DB
	log *logger.Logger
}

func NewGPTService(db *sqlx.DB) *gptService {
	return &gptService{
		DB:  db,
		log: logger.New("gptService"),
	}
}

func (db *gptService) GetResponseID(chat *gotgbot.Chat) (model.GPTData, error) {
	const query = `SELECT gpt_response_id, gpt_response_id_expires_on FROM chats WHERE id = ?`
	var data model.GPTData
	err := db.Get(&data, query, chat.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return data, nil
		}
	}
	return data, err
}

func (db *gptService) ResetResponseID(chat *gotgbot.Chat) error {
	const query = `UPDATE chats SET gpt_response_id = NULL, gpt_response_id_expires_on = NULL WHERE id = ?`
	_, err := db.Exec(query, chat.Id)
	return err
}

func (db *gptService) SetResponseID(chat *gotgbot.Chat, responseID string) error {
	const query = `UPDATE chats
	SET gpt_response_id = ?,
	    gpt_response_id_expires_on = NOW() + INTERVAL 30 MINUTE
	WHERE id = ?`
	_, err := db.Exec(query, responseID, chat.Id)
	return err
}
