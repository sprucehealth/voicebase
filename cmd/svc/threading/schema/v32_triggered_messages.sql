CREATE TABLE triggered_messages (
   id BIGINT UNSIGNED NOT NULL,
   actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
   organization_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
   trigger_key VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL, -- key is a protected word
   trigger_subkey VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL, -- prefixed for consistency
   enabled BOOL NOT NULL,
   created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
   modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
   PRIMARY KEY (id),
   UNIQUE uc_org_entity_id_trigger_key_trigger_subkey (organization_entity_id, trigger_key, trigger_subkey),
   KEY idx_owner_entity_id (actor_entity_id)
);

CREATE TABLE triggered_message_items (
   id BIGINT UNSIGNED NOT NULL,
   triggered_message_id BIGINT UNSIGNED NOT NULL,
   actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
   ordinal INT NOT NULL,
   internal BOOLEAN NOT NULL,
   type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
   data BLOB NOT NULL, -- serialized protocol buffer with structure based on the type of the item,
   created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
   modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
   PRIMARY KEY (id),
   CONSTRAINT fk_triggered_message_id_triggered_messages_id FOREIGN KEY (triggered_message_id) REFERENCES triggered_messages (id)
);