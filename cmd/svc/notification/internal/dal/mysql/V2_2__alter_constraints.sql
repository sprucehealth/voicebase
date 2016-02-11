-- drop the existing unique constraint
ALTER TABLE push_config DROP INDEX external_group_id;

-- Create an index just on the device token
CREATE UNIQUE INDEX idx_device_token ON push_config (device_token);