CREATE TABLE incoming_call (
	source VARCHAR(16) NOT NULL,
	destination VARCHAR(16) NOT NULL,
	organization_id varchar(80) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	call_sid VARCHAR(36),
	PRIMARY KEY (call_sid)
) CHARACTER SET ascii COLLATE ascii_bin;