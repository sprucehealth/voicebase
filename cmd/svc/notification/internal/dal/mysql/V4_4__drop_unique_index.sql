-- drop unique index on device_token given that the same device_token could appear twice
-- (if deleted and then recreated)
ALTER TABLE push_config DROP INDEX idx_device_token;

-- Add column to soft delete push configs
ALTER TABLE push_config ADD COLUMN deleted TINYINT(1) NOT NULL DEFAULT 0;

-- drop index on device_id to recreate as composite index with deleted column.
ALTER TABLE push_config DROP INDEX idx_device_id;

-- Create indexes on the colums used to access push configs

CREATE INDEX idx_device_id ON push_config (device_id, deleted);

CREATE INDEX idx_device_token ON push_config (device_token, deleted);

CREATE INDEX idx_external_group_id ON push_config (external_group_id, deleted);

