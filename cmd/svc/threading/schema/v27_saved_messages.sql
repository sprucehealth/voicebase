CREATE TABLE saved_message (
    id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(2048) NOT NULL,
    organization_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    owner_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    creator_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    internal BOOLEAN NOT NULL,
    type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    data BLOB NOT NULL, -- serialized protocol buffer with structure based on the type of the item
    created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY owner_entity_id (owner_entity_id)
);
