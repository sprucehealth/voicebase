ALTER TABLE proxy_phone_number DROP KEY expires;
ALTER TABLE proxy_phone_number DROP COLUMN expires;
ALTER TABLE proxy_phone_number DROP COLUMN last_reserved_time;

-- drop the table to cleanly start from the top.
DROP TABLE proxy_phone_number_reservation;

CREATE TABLE proxy_phone_number_reservation (
	proxy_phone_number VARCHAR(16) NOT NULL,
	originating_phone_number VARCHAR(16) NOT NULL,
	destination_phone_number VARCHAR(16) NOT NULL,
	destination_entity_id VARCHAR(64) NOT NULL,
	owner_entity_id VARCHAR(64) NOT NULL,
	organization_id VARCHAR(64) NOT NULL,
	expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	KEY (originating_phone_number, proxy_phone_number, destination_phone_number, expires)
) CHARACTER SET ascii COLLATE ascii_bin;

-- this table keeps track of the current originating phone number for each entity 
CREATE TABLE originating_phone_number (
	phone_number VARCHAR(16) NOT NULL,
	entity_id VARCHAR(64) NOT NULL,
	PRIMARY KEY(entity_id)
) CHARACTER SET ascii COLLATE ascii_bin;	