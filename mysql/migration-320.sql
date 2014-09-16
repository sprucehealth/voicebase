ALTER TABLE account ADD COLUMN two_factor_enabled BOOL NOT NULL DEFAULT false;

CREATE TABLE account_device (
    account_id int unsigned NOT NULL,
    device_id varchar(128) NOT NULL,
    verified BOOL NOT NULL,
    verified_tstamp timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES account (id),
    PRIMARY KEY (account_id, device_id)
) character set utf8mb4;
