-- +migrate Up

CREATE TABLE chats
(
    id         BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NULL     DEFAULT NULL,
    title      VARCHAR(255) NOT NULL,
    allowed    BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE INDEX ON chats (allowed);

CREATE TABLE users
(
    id         BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ  NULL     DEFAULT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name  VARCHAR(255) NULL,
    username   VARCHAR(255) NULL,
    allowed    BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE INDEX ON users (allowed);
CREATE INDEX ON users (username);

CREATE TABLE chats_users
(
    chat_id    BIGINT,
    user_id    BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    msg_count  BIGINT      NOT NULL DEFAULT 1,
    in_group   BOOLEAN     NOT NULL DEFAULT TRUE,
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

CREATE TABLE plugins
(
    name       VARCHAR(25) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NULL     DEFAULT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE
);

CREATE INDEX ON plugins (enabled);

CREATE TABLE chats_plugins
(
    chat_id     BIGINT,
    plugin_name VARCHAR(25),
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


CREATE TABLE credentials
(
    name       VARCHAR(50) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    value      TEXT        NOT NULL
);
