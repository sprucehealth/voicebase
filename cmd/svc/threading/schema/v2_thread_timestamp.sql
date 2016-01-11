ALTER TABLE threads ADD COLUMN last_message_timestamp TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);
UPDATE threads SET last_message_timestamp = (SELECT MAX(created) FROM thread_items WHERE thread_id = threads.id);
CREATE INDEX threads_org_last_timestamp ON threads (organization_id, last_message_timestamp);

ALTER TABLE threads ADD COLUMN last_external_message_timestamp TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);
UPDATE threads SET last_external_message_timestamp = (SELECT MAX(created) FROM thread_items WHERE thread_id = threads.id AND internal = false);
CREATE INDEX threads_org_last_external_timestamp ON threads (organization_id, last_external_message_timestamp);
