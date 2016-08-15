-- sync_config stores the sync configuration for a given organization
CREATE TABLE sync_config (
	org_id VARCHAR(64) NOT NULL,
	config BLOB NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (org_id)
);