-- +migrate Up

CREATE TABLE `quotes`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp(),
    `chat_id`    BIGINT(20)          NOT NULL,
    `quote`      LONGTEXT            NOT NULL,
    CONSTRAINT `FK_quotes_chats` FOREIGN KEY (`chat_id`) REFERENCES `chats` (`id`) ON UPDATE CASCADE ON DELETE CASCADE,
    INDEX `quote` (`quote`(768))
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;