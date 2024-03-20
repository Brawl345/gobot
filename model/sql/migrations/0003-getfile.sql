-- +migrate Up

CREATE TABLE files
(
    id         VARCHAR(128) PRIMARY KEY,
    file_name  TEXT        NOT NULL,
    type       VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);