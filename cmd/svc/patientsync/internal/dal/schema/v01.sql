-- sync_config stores the sync configuration for a given organization
CREATE TABLE sync_config (
	org_id VARCHAR(64) NOT NULL,
	source VARCHAR(64) CHARSET ascii COLLATE ascii_bin NOT NULL,
	external_id VARCHAR(64) CHARSET ascii COLLATE ascii_bin,
	config BLOB NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (org_id),
	KEY (source),
	UNIQUE KEY uk_external_id (external_id)
);

CREATE TABLE sync_bookmark (
	org_id VARCHAR(64) NOT NULL,
	bookmark TIMESTAMP NOT NULL,
	status VARCHAR(64) NOT NULL,
	PRIMARY KEY (org_id)
);
