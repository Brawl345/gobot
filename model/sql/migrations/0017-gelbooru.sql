-- +migrate Up

CREATE TABLE `gelbooru_queries`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp(),
    `query`      VARCHAR(2048)        NOT NULL,
    INDEX `query` (`query`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;
