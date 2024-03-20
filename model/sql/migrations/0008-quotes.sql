-- +migrate Up

CREATE TABLE quotes
(
    id         SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    chat_id    BIGINT      NOT NULL,
    quote      TEXT        NOT NULL,
    CONSTRAINT fk_quotes_chats FOREIGN KEY (chat_id)
        REFERENCES chats (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX ON quotes (quote);
