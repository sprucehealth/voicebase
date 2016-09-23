-- Backfill everything to enabled
ALTER TABLE saved_queries ADD COLUMN notifications_enabled BOOL NOT NULL DEFAULT TRUE;

-- Change the default
ALTER TABLE saved_queries MODIFY COLUMN notifications_enabled BOOL NOT NULL DEFAULT FALSE;

-- Disable All queries
UPDATE saved_queries SET notifications_enabled = FALSE WHERE title = 'All';