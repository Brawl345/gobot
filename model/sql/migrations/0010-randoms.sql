-- +migrate Up

CREATE TABLE randoms
(
    id         SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    text       TEXT        NOT NULL
);

CREATE INDEX ON randoms (text);