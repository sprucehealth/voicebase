
CREATE TABLE thread_links (
    thread1_id BIGINT UNSIGNED NOT NULL,
    thread2_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (thread1_id),
    UNIQUE KEY thread2_id (thread2_id),
    CONSTRAINT thread_links_thread1_id FOREIGN KEY (thread1_id) REFERENCES threads (id),
    CONSTRAINT thread_links_thread2_id FOREIGN KEY (thread2_id) REFERENCES threads (id)
);

CREATE TABLE onboarding_threads (
    thread_id BIGINT UNSIGNED NOT NULL,
    entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    step INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (thread_id),
    UNIQUE KEY (entity_id),
    CONSTRAINT onboarding_threads_thread_id FOREIGN KEY (thread_id) REFERENCES threads (id)
);
