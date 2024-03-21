-- +migrate Up

CREATE TABLE randoms
(
    id         INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    text       TEXT        NOT NULL
);

CREATE INDEX ON randoms (text);