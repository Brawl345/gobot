-- +migrate Up

INSERT INTO `plugins` (`name`, `enabled`)
VALUES ('about', 1),
       ('allow', 1),
       ('creds', 1),
       ('echo', 1),
       ('id', 1),
       ('manager', 1),
       ('stats', 1);
