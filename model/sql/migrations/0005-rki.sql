-- +migrate Up

ALTER TABLE  `users`
    ADD `rki_ags` VARCHAR(8) NULL;