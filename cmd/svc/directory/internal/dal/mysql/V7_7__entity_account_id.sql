-- Add support for tracking account_id
ALTER TABLE entity ADD COLUMN account_id VARCHAR(64) NOT NULL;
ALTER TABLE entity ADD UNIQUE (account_id, id);

-- Backfill the account info
UPDATE entity JOIN external_entity_id ON entity.id = external_entity_id.entity_id SET account_id = external_entity_id.external_id;