-- +migrate Up

CREATE TABLE `brave_images_queries`
(
    `id`            INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at`    DATETIME            NOT NULL DEFAULT current_timestamp(),
    `query`         VARCHAR(512)        NOT NULL,
    `current_index` TINYINT(1)          NOT NULL DEFAULT 1,
    INDEX `query` (`query`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `brave_images`
(
    `id`          INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at`  DATETIME            NOT NULL DEFAULT current_timestamp(),
    `query_id`    INT(11)             NOT NULL,
    `image_url`   VARCHAR(2048)       NOT NULL,
    `context_url` VARCHAR(2048)       NOT NULL,
    `is_gif`      TINYINT(1)          NOT NULL DEFAULT 0,
    FOREIGN KEY (`query_id`) REFERENCES `brave_images_queries` (`id`) ON UPDATE CASCADE ON DELETE CASCADE
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;