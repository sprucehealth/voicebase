CREATE TABLE ipcall (
    id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    pending BOOL NOT NULL,
    initiated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE ipcall_participant (
    ipcall_id BIGINT UNSIGNED NOT NULL,
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    entity_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    identity VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    role VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    state VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    UNIQUE (ipcall_id, account_id),
    CONSTRAINT ipcall_participant_ipcall_id FOREIGN KEY (ipcall_id) REFERENCES ipcall (id),
    KEY (account_id)
);
