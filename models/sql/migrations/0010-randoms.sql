-- +migrate Up

CREATE TABLE `randoms`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp(),
    `text`      LONGTEXT            NOT NULL,
    INDEX `text` (`text`(768))
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;