-- +migrate Up

-- +migrate StatementBegin
CREATE FUNCTION trigger_set_updated_at()
    RETURNS TRIGGER AS
$$
BEGIN
    IF NEW IS DISTINCT FROM OLD THEN
        NEW.updated_at = CLOCK_TIMESTAMP();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TABLE chats
(
    id                             BIGINT PRIMARY KEY,
    created_at                     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                     TIMESTAMPTZ NULL     DEFAULT NULL,
    title                          TEXT        NOT NULL,
    allowed                        BOOLEAN     NOT NULL DEFAULT FALSE,
    cleverbot_state                TEXT        NULL,
    birthday_notifications_enabled BOOLEAN     NOT NULL DEFAULT FALSE,
    gemini_history                 JSONB       NULL,
    gemini_history_expires_on      TIMESTAMPTZ NULL
);

CREATE INDEX ON chats (allowed);
CREATE INDEX ON chats (birthday_notifications_enabled);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON chats
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE users
(
    id         BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    first_name TEXT        NOT NULL,
    last_name  TEXT        NULL,
    username   TEXT        NULL,
    allowed    BOOLEAN     NOT NULL DEFAULT FALSE,
    birthday   DATE        NULL
);

CREATE INDEX ON users (allowed);
CREATE INDEX ON users (username);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE chats_users
(
    chat_id    BIGINT,
    user_id    BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    msg_count  BIGINT      NOT NULL DEFAULT 1,
    in_group   BOOLEAN     NOT NULL DEFAULT TRUE,
    notify     BOOLEAN              DEFAULT FALSE,
    afk_since  TIMESTAMPTZ NULL,
    afk_reason TEXT        NULL,
    PRIMARY KEY (chat_id, user_id),
    CONSTRAINT fk_chats_users_chats FOREIGN KEY (chat_id)
        REFERENCES chats (id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    CONSTRAINT fk_chats_users_users FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX ON chats_users (user_id);
CREATE INDEX ON chats_users (in_group);
CREATE INDEX ON chats_users (notify);
CREATE INDEX ON chats_users (afk_since);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON chats_users
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE plugins
(
    name       TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE
);

CREATE INDEX ON plugins (enabled);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON plugins
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE chats_plugins
(
    chat_id     BIGINT,
    plugin_name TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NULL     DEFAULT NULL,
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    PRIMARY KEY (chat_id, plugin_name),
    CONSTRAINT fk_chats_plugins_chats FOREIGN KEY (chat_id)
        REFERENCES chats (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT fk_chats_plugins_plugins FOREIGN KEY (plugin_name)
        REFERENCES plugins (name)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX ON chats_plugins (plugin_name);
CREATE INDEX ON chats_plugins (enabled);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON chats_plugins
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();

CREATE TABLE credentials
(
    name       TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    value      TEXT        NOT NULL
);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE
    ON credentials
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_updated_at();
