-- ALTER TABLE thread_entities ADD COLUMN unread_reference BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE thread_entities ADD COLUMN last_referenced TIMESTAMP NULL;
