-- +migrate Up

ALTER TABLE `chats_users`
    ADD `afk_since`  DATETIME NULL,
    ADD `afk_reason` TEXT     NULL,
    ADD INDEX `afk_since` (`afk_since`);
