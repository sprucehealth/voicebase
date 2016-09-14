UPDATE thread_entities te
INNER JOIN threads t ON t.id = te.thread_id
SET te.last_viewed = t.last_message_timestamp
WHERE te.last_viewed IS NOT NULL AND date_format(last_message_timestamp, '%Y-%m-%d %H:%i:%S') = te.last_viewed;
