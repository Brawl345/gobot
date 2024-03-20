-- +migrate Up

INSERT INTO plugins (name, enabled)
VALUES ('about', TRUE),
       ('alive', TRUE),
       ('allow', TRUE),
       ('creds', TRUE),
       ('echo', TRUE),
       ('id', TRUE),
       ('manager', TRUE),
       ('stats', TRUE);
