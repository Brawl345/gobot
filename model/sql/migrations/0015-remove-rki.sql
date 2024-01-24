-- +migrate Up

ALTER TABLE `users`
    DROP COLUMN `rki_ags`;
