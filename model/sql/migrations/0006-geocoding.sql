-- +migrate Up

CREATE TABLE geocoding
(
    id         SERIAL PRIMARY KEY,
    address    VARCHAR(512)   NOT NULL,
    latitude   DECIMAL(10, 7) NOT NULL,
    longitude  DECIMAL(10, 7) NOT NULL,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (latitude, longitude)
);

ALTER TABLE users
    ADD COLUMN home INT NULL,
    ADD CONSTRAINT fk_users_geocoding
        FOREIGN KEY (home)
            REFERENCES geocoding (id)
            ON DELETE CASCADE
            ON UPDATE CASCADE;

