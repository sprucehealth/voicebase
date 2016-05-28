CREATE TABLE auth.login_info (
  account_id bigint UNSIGNED NOT NULL,
  device_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  platform varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  last_login_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  primary key (account_id, platform),
  KEY (last_login_timestamp)
);
