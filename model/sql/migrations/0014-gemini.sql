-- +migrate Up

ALTER TABLE chats
    ADD gemini_history            TEXT        NULL,
    ADD gemini_history_expires_on TIMESTAMPTZ NULL;