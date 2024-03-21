-- +migrate Up

CREATE TABLE files
(
    id         TEXT PRIMARY KEY,
    file_name  TEXT        NOT NULL,
    type       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);