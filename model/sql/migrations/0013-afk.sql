-- +migrate Up

ALTER TABLE chats_users
    ADD afk_since  TIMESTAMPTZ NULL,
    ADD afk_reason TEXT        NULL;

CREATE INDEX ON chats_users (afk_since);