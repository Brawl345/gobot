-- +migrate Up

CREATE TABLE `reminders`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp(),
    `chat_id`    BIGINT(20)          NULL,
    `user_id`    BIGINT(20)          NOT NULL,
    `time`       DATETIME            NOT NULL,
    `text`       LONGTEXT            NOT NULL,
    CONSTRAINT `FK_randoms_chats` FOREIGN KEY (`chat_id`) REFERENCES `chats` (`id`) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT `FK_randoms_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE CASCADE,
    INDEX `text` (`text`(768))
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;