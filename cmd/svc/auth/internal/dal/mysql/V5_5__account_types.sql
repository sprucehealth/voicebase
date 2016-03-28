ALTER TABLE account ADD COLUMN type VARCHAR(25) NOT NULL DEFAULT 'PROVIDER';
-- TODO: mraines: Remove the default value after the required changes have been deployed