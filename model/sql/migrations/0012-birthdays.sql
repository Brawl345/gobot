-- +migrate Up

ALTER TABLE `users`
    ADD `birthday` DATE NULL;

ALTER TABLE `chats`
    ADD `birthday_notifications_enabled` TINYINT(1) NOT NULL DEFAULT '0',
    ADD INDEX `birthday_notifications_enabled` (`birthday_notifications_enabled`);