CREATE TABLE scheduled_messages (
    id BIGINT UNSIGNED NOT NULL,
	thread_id BIGINT UNSIGNED NOT NULL,
    actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    internal BOOLEAN NOT NULL,
    type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    data BLOB NOT NULL, -- serialized protocol buffer with structure based on the type of the item
	status varchar(50) NOT NULL,
	scheduled_for TIMESTAMP(6) NOT NULL,
	sent_at TIMESTAMP(6),
    sent_thread_item_id BIGINT UNSIGNED,
    created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
	CONSTRAINT fk_thread_id_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id),
    CONSTRAINT fk_sent_thread_item_id_thread_item_id FOREIGN KEY (sent_thread_item_id) REFERENCES thread_items (id),
    KEY idx_actor_entity_id (actor_entity_id),
	KEY idx_status_scheduled_for (status, scheduled_for)
);