-- +migrate Up

ALTER TABLE  `chats`
    ADD `cleverbot_state` MEDIUMTEXT NULL;
