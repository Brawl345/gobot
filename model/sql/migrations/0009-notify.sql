-- +migrate Up

ALTER TABLE chats_users
    ADD COLUMN notify BOOLEAN DEFAULT FALSE;

CREATE INDEX ON chats_users (notify);