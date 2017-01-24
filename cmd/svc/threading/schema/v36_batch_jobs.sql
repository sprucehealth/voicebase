-- track sets of batch tasks
CREATE TABLE batch_jobs (
    id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    requesting_entity VARCHAR(150) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    tasks_requested INT UNSIGNED NOT NULL,
    tasks_completed INT UNSIGNED NOT NULL,
    tasks_errored INT UNSIGNED NOT NULL,
    completed TIMESTAMP(6),
    created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_completed (completed)
);

-- track individual sets of work
CREATE TABLE batch_tasks (
    id BIGINT UNSIGNED NOT NULL,
    batch_job_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    type VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    error VARCHAR(256),
    data BLOB NOT NULL, -- serialized protocol buffer with structure based on the type of the item
    available_after TIMESTAMP(6) NOT NULL,
    completed TIMESTAMP(6),
    created TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    modified TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_status_available_after (status, available_after),
    CONSTRAINT fk_batch_job_batch_job_id FOREIGN KEY (batch_job_id) REFERENCES batch_jobs (id)
);