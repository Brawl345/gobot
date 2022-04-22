-- +migrate Up

CREATE TABLE `geocoding`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `address`    VARCHAR(512)        NOT NULL,
    `latitude`   DECIMAL(10, 7)      NOT NULL,
    `longitude`  DECIMAL(10, 7)      NOT NULL,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp()
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `geocoding_queries`
(
    `id`           INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `query`        VARCHAR(512)        NOT NULL,
    `geocoding_id` INT(11)             NOT NULL,
    CONSTRAINT `fk_geocoding_queries_geocoding`
        FOREIGN KEY (`geocoding_id`)
            REFERENCES `geocoding` (`id`)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    INDEX `query` (`query`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

ALTER TABLE `users`
    ADD `home` INT(11) NULL,
    ADD CONSTRAINT `fk_geocoding_queries_geocoding`
        FOREIGN KEY (`home`)
            REFERENCES `geocoding` (`id`)
            ON DELETE CASCADE
            ON UPDATE CASCADE;
