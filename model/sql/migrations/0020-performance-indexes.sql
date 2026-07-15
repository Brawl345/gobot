-- +migrate Up

ALTER TABLE `quotes`
    ADD INDEX `chat_id_quote` (`chat_id`, `quote`(700)),
    DROP INDEX `quote`,
    DROP INDEX `chat_id`;

ALTER TABLE `google_images_queries`
    ADD INDEX `created_at` (`created_at`);

ALTER TABLE `brave_images_queries`
    ADD INDEX `created_at` (`created_at`);

ALTER TABLE `gelbooru_queries`
    ADD INDEX `created_at` (`created_at`);
