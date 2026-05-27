-- +migrate Up

ALTER TABLE `chats`
    ADD `gpt_response_id`            VARCHAR(255) NULL,
    ADD `gpt_response_id_expires_on` DATETIME     NULL;
