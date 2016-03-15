-- Keep a counter of the number of messages that have been posted to a thread
ALTER TABLE threads ADD COLUMN message_count INT UNSIGNED DEFAULT 0 NOT NULL;