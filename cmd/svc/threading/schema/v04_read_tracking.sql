ALTER TABLE thread_members ADD COLUMN last_viewed TIMESTAMP;

CREATE TABLE thread_item_view_details (
    thread_item_id BIGINT UNSIGNED NOT NULL,
    actor_entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    view_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (thread_item_id, actor_entity_id),
    CONSTRAINT thread_items_thread_item_id FOREIGN KEY (thread_item_id) REFERENCES thread_items (id)
);