CREATE TABLE threads (
    id BIGINT UNSIGNED NOT NULL,
    organization_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    primary_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin,
    PRIMARY KEY (id),
    KEY organization_id (organization_id),
    KEY primary_entity_id (primary_entity_id)
);

CREATE TABLE thread_members (
    thread_id BIGINT UNSIGNED NOT NULL,
    entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    following BOOLEAN NOT NULL DEFAULT false,
    joined TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (thread_id, entity_id),
    KEY entity_id (entity_id),
    CONSTRAINT thread_members_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id)
);

CREATE TABLE thread_items (
    id BIGINT UNSIGNED NOT NULL,
    thread_id BIGINT UNSIGNED NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    internal BOOLEAN NOT NULL,
    type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    data BLOB NOT NULL, -- serialized protocol buffer with structure based on the type of the item
    PRIMARY KEY (id),
    KEY thread_id (thread_id),
    CONSTRAINT thread_items_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id)
);

CREATE TABLE saved_queries (
    id BIGINT UNSIGNED NOT NULL,
    organization_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    query BLOB NOT NULL, -- serialized protocol buffer
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY entity_organization (entity_id, organization_id)
);
