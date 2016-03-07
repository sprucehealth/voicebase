-- Map the client encryption key into the auth token
ALTER TABLE auth_token ADD COLUMN client_encryption_key BLOB;
ALTER TABLE auth_token ADD COLUMN shadow BOOL NOT NULL DEFAULT FALSE;

-- Track 2FA login  attempts
CREATE TABLE auth.two_factor_login (
    account_id          BIGINT UNSIGNED NOT NULL,
	device_id           VARCHAR(100)  NOT NULL,
    last_login          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, device_id)
) engine=InnoDB;