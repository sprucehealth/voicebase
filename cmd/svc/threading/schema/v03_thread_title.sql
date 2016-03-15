ALTER TABLE threads ADD COLUMN last_message_summary VARCHAR(1024);
ALTER TABLE threads ADD COLUMN last_external_message_summary VARCHAR(1024);
UPDATE threads SET last_message_summary = primary_entity_id;
UPDATE threads SET last_external_message_summary = primary_entity_id;
ALTER TABLE threads MODIFY COLUMN last_message_summary VARCHAR(1024) NOT NULL;
ALTER TABLE threads MODIFY COLUMN last_external_message_summary VARCHAR(1024) NOT NULL;
