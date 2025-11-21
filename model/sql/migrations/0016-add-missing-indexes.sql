-- +migrate Up

ALTER TABLE `quotes`
    ADD INDEX `chat_id` (`chat_id`);

ALTER TABLE `reminders`
    ADD INDEX `time` (`time`);

ALTER TABLE `reminders`
    ADD INDEX `chat_id_time` (`chat_id`, `time`);

ALTER TABLE `reminders`
    ADD INDEX `user_id_time` (`user_id`, `time`);
