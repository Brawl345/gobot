-- +migrate Up

CREATE TABLE google_images_queries
(
    id            SERIAL PRIMARY KEY,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    query         VARCHAR(512) NOT NULL,
    current_index SMALLINT     NOT NULL DEFAULT 1
);

CREATE INDEX ON google_images_queries (query);

CREATE TABLE google_images
(
    id          SERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    query_id    INT           NOT NULL,
    image_url   VARCHAR(2048) NOT NULL,
    context_url VARCHAR(2048) NOT NULL,
    is_gif      BOOLEAN       NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_google_images_google_images_queries
        FOREIGN KEY (query_id)
            REFERENCES google_images_queries (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE
);
