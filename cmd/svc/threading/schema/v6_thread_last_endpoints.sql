-- Add column to hold the last routes associated with the primary entities interaction with the thread
ALTER TABLE threads ADD COLUMN last_primary_entity_endpoints BLOB;