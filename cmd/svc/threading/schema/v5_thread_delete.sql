ALTER TABLE threads ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE thread_events (
    id BIGINT UNSIGNED NOT NULL,
    thread_id BIGINT UNSIGNED NOT NULL,
    time TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    event VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT thread_events_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id)
);

ALTER TABLE threads MODIFY COLUMN last_message_summary VARCHAR(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL;
ALTER TABLE threads MODIFY COLUMN last_external_message_summary VARCHAR(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL;

DROP INDEX threads_org_last_timestamp ON threads;
DROP INDEX threads_org_last_external_timestamp ON threads;

CREATE INDEX threads_org_deleted_last_timestamp ON threads (organization_id, deleted, last_message_timestamp);
CREATE INDEX threads_org_deleted_last_external_timestamp ON threads (organization_id, deleted, last_external_message_timestamp);
