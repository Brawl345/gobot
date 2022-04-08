-- +migrate Up

CREATE TABLE `chats`
(
    `id`         BIGINT(20) NOT NULL PRIMARY KEY,
    `created_at` DATETIME   NOT NULL DEFAULT current_timestamp(),
    `updated_at` DATETIME   NULL     DEFAULT NULL ON UPDATE current_timestamp(),
    `title`      TINYTEXT   NOT NULL,
    `allowed`    TINYINT(1) NOT NULL DEFAULT 0,
    INDEX `allowed` (`allowed`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `users`
(
    `id`         BIGINT(20) NOT NULL PRIMARY KEY,
    `created_at` DATETIME   NOT NULL DEFAULT current_timestamp(),
    `updated_at` DATETIME   NULL     DEFAULT NULL ON UPDATE current_timestamp(),
    `first_name` TINYTEXT   NOT NULL,
    `last_name`  TINYTEXT   NULL,
    `username`   TINYTEXT   NULL,
    `allowed`    TINYINT(1) NOT NULL DEFAULT 0,
    INDEX `allowed` (`allowed`),
    INDEX `username` (`username`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `chats_users`
(
    `chat_id`    BIGINT(20) NOT NULL,
    `user_id`    BIGINT(20) NOT NULL,
    `created_at` DATETIME   NOT NULL DEFAULT current_timestamp(),
    `updated_at` DATETIME   NULL     DEFAULT NULL ON UPDATE current_timestamp(),
    `msg_count`  BIGINT     NOT NULL DEFAULT 1,
    `in_group`   TINYINT(1) NOT NULL DEFAULT 1,
    PRIMARY KEY (`chat_id`, `user_id`),
    INDEX `FK_chats_users_users` (`user_id`),
    INDEX `in_group` (`in_group`),
    CONSTRAINT `FK_chats_users_chats` FOREIGN KEY (`chat_id`) REFERENCES `chats` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT `FK_chats_users_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE CASCADE ON DELETE CASCADE
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `plugins`
(
    `name`       VARCHAR(25) NOT NULL PRIMARY KEY,
    `created_at` DATETIME    NOT NULL DEFAULT current_timestamp(),
    `updated_at` DATETIME    NULL     DEFAULT NULL ON UPDATE current_timestamp(),
    `enabled`    TINYINT(1)  NOT NULL DEFAULT 1,
    INDEX `enabled` (`enabled`)
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `chats_plugins`
(
    `chat_id`     BIGINT(20)  NOT NULL,
    `plugin_name` VARCHAR(25) NOT NULL,
    `created_at`  DATETIME    NOT NULL DEFAULT current_timestamp(),
    `updated_at`  DATETIME    NULL     DEFAULT NULL ON UPDATE current_timestamp(),
    `enabled`     TINYINT(4)  NOT NULL DEFAULT 1,
    PRIMARY KEY (`chat_id`, `plugin_name`),
    INDEX `FK_chats_plugins_plugins` (`plugin_name`),
    INDEX `enabled` (`enabled`),
    CONSTRAINT `FK_chats_plugins_chats` FOREIGN KEY (`chat_id`) REFERENCES `andi_gobot`.`chats` (`id`) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT `FK_chats_plugins_plugins` FOREIGN KEY (`plugin_name`) REFERENCES `andi_gobot`.`plugins` (`name`) ON UPDATE CASCADE ON DELETE CASCADE
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;

CREATE TABLE `credentials`
(
    `name`  VARCHAR(50)  NOT NULL PRIMARY KEY,
    `value` VARCHAR(255) NOT NULL
) COLLATE = 'utf8mb4_general_ci'
  ENGINE = InnoDB;
