-- +migrate Up

ALTER TABLE `chats`
    ADD `gemini_history`            MEDIUMTEXT NULL,
    ADD `gemini_history_expires_on` DATETIME   NULL;