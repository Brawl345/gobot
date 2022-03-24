-- +migrate Up

CREATE TABLE `files`
(
    `id`         VARCHAR(128) NOT NULL PRIMARY KEY,
    `file_name`  TEXT         NOT NULL,
    `type`       VARCHAR(20)  NOT NULL,
    `created_at` DATETIME     NOT NULL DEFAULT current_timestamp()
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;