-- +migrate Up

INSERT INTO `plugins` (`name`, `enabled`)
VALUES ('about', 1),
       ('echo', 1),
       ('manager', 1),
       ('stats', 1);
