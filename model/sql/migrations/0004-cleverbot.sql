-- +migrate Up

ALTER TABLE chats
    ADD cleverbot_state TEXT NULL;
