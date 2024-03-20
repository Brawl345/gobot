-- +migrate Up

CREATE TABLE reminders
(
    id         SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    chat_id    BIGINT      NULL,
    user_id    BIGINT      NOT NULL,
    time       TIMESTAMPTZ NOT NULL,
    text       TEXT        NOT NULL,
    CONSTRAINT fk_reminders_chats FOREIGN KEY (chat_id)
        REFERENCES chats (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT fk_reminders_users FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX ON reminders (text);