-- +migrate Up

-- Add index on quotes.chat_id for better filtering performance
-- All quote queries filter by chat_id
ALTER TABLE `quotes`
    ADD INDEX `chat_id` (`chat_id`);

-- Add index on reminders.time for ORDER BY operations
ALTER TABLE `reminders`
    ADD INDEX `time` (`time`);

-- Add composite index on reminders(chat_id, time)
-- Optimizes: SELECT ... WHERE chat_id = ? ORDER BY time
ALTER TABLE `reminders`
    ADD INDEX `chat_id_time` (`chat_id`, `time`);

-- Add composite index on reminders(user_id, time)
-- Optimizes: SELECT ... WHERE chat_id IS NULL AND user_id = ? ORDER BY time
ALTER TABLE `reminders`
    ADD INDEX `user_id_time` (`user_id`, `time`);
