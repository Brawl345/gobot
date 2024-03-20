-- +migrate Up

ALTER TABLE users
    ADD COLUMN "birthday" DATE NULL;

ALTER TABLE chats
    ADD COLUMN "birthday_notifications_enabled" BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX ON chats (birthday_notifications_enabled);
