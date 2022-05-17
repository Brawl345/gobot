-- +migrate Up

ALTER TABLE  `chats_users`
    ADD `notify` TINYINT(1) DEFAULT 0,
    ADD INDEX `notify` (`notify`);