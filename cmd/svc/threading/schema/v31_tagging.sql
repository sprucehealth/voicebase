CREATE TABLE tags (
    id INT NOT NULL AUTO_INCREMENT,
    hidden BOOLEAN NOT NULL,
    organization_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    tag VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY (organization_id, tag)
);

CREATE TABLE thread_tags (
    thread_id BIGINT UNSIGNED NOT NULL,
    tag_id INT NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (thread_id, tag_id),
    CONSTRAINT thread_tags_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id) ON DELETE CASCADE,
    CONSTRAINT thread_tags_tag_id FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE
);
