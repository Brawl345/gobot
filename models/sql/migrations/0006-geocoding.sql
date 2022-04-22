-- +migrate Up

CREATE TABLE `geocoding`
(
    `id`         INT(11) PRIMARY KEY NOT NULL AUTO_INCREMENT,
    `address`    VARCHAR(512)        NOT NULL,
    `latitude`   DECIMAL(10, 7)       NOT NULL,
    `longitude`  DECIMAL(10, 7)       NOT NULL,
    `created_at` DATETIME            NOT NULL DEFAULT current_timestamp(),
    UNIQUE INDEX `latitude_longitude` (`latitude`, `longitude`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

ALTER TABLE `users`
    ADD `home` INT(11) NULL,
    ADD CONSTRAINT `FK_users_geocoding`
        FOREIGN KEY (`home`)
            REFERENCES `geocoding` (`id`)
            ON DELETE CASCADE
            ON UPDATE CASCADE;
