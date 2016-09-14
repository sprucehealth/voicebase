ALTER TABLE thread_entities MODIFY last_viewed TIMESTAMP(6) NULL DEFAULT NULL;
ALTER TABLE thread_entities MODIFY last_unread_notify TIMESTAMP(6) NULL DEFAULT NULL;
ALTER TABLE thread_entities MODIFY last_referenced TIMESTAMP(6) NULL DEFAULT NULL;

-- If last referenced time is after last message then fix it up as that's impossible.
UPDATE thread_entities te
INNER JOIN threads t ON t.id = te.thread_id
SET te.last_referenced = t.last_message_timestamp
WHERE te.last_referenced IS NOT NULL AND te.last_referenced > t.last_message_timestamp;
