-- excomms_event represents a historical occurence of an external communication.
CREATE TABLE excomms_event (
	source VARCHAR(256) NOT NULL, 
	destination VARCHAR(256) NOT NULL,
	data blob NOT NULL,
	event VARCHAR(256) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- provisioned_phone_number represents a list of phone numbers
-- that have been provisioned.
CREATE TABLE provisioned_phone_number (
	phone_number VARCHAR(16) NOT NULL,
	provisioned_for VARCHAR(64) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE KEY (phone_number),
	UNIQUE KEY (provisioned_for)
);

-- outgoing_call_request represents a request by a provider
-- to make a call from a source to a destination via a proxy number.
CREATE TABLE outgoing_call_request (
	source VARCHAR(16),
	destination VARCHAR(16) NOT NULL,
	proxy VARCHAR(16) NOT NULL,
	organization_id varchar(80) NOT NULL,
	requested TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	call_sid VARCHAR(36),
	expires TIMESTAMP NULL,
	KEY (source, expires)
);
