-- Index our device ids so we can quickly unregister devices
CREATE INDEX idx_device_id ON push_config (device_id);