-- proxy_phone_number contains a list of phone numbers that are 
-- either available or reserved for outgoing phone calls.
CREATE TABLE proxy_phone_number (
	phone_number VARCHAR(16) NOT NULL,
	expires TIMESTAMP,
	KEY (expires),
	PRIMARY KEY (phone_number)
);

-- proxy_phone_number_reservation contains an entry of every phone
-- number reservation.
CREATE TABLE proxy_phone_number_reservation (
	phone_number VARCHAR(16) NOT NULL,
	destination_entity_id VARCHAR(64) NOT NULL,
	owner_entity_id VARCHAR(64) NOT NULL,
	organization_id VARCHAR(64) NOT NULL,
	expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	KEY (phone_number, expires),
	KEY (destination_entity_id, expires)
);

-- Also drop the expires column as that will be managed by the proxy_phone_number_reservation.
ALTER TABLE outgoing_call_request DROP COLUMN expires;
