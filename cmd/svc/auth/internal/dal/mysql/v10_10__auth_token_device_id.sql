ALTER TABLE auth_token ADD COLUMN device_id VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE auth_token ADD COLUMN platform VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE auth_token DROP INDEX account_shadow_expires;
CREATE INDEX idx_account_shadow_expires_device_id ON auth_token (account_id, shadow, expires, device_id);
CREATE INDEX idx_account_shadow_expires_duration_type ON auth_token (account_id, shadow, expires, duration_type);