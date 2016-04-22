-- Rename the table to apply for all generic endpoints (phone, email, etc)
RENAME TABLE provisioned_phone_number  TO provisioned_endpoint;

-- Increase the length of the endpoint so that it can store the maximum length of an email which is 254 bytes.
ALTER TABLE provisioned_endpoint CHANGE COLUMN phone_number endpoint VARCHAR(255) NOT NULL;

-- Introduce type to the endpoint so that we can identify that we can identify the type of each endpoint.
ALTER TABLE provisioned_endpoint ADD COLUMN endpoint_type VARCHAR(32) NOT NULL DEFAULT 'phone';

-- Drop the unique constraint on provisioned_for to then create the unique key constraint on the (provisioned_for, endpoint_type) combination.
ALTER TABLE provisioned_endpoint DROP INDEX provisioned_for;
ALTER TABLE provisioned_endpoint ADD UNIQUE KEY (provisioned_for, endpoint_type);


-- sent_messaage persists messages that were sent by the excomms 
-- service.
CREATE TABLE sent_message (
	id BIGINT UNSIGNED NOT NULL,
	uuid VARCHAR(64) NOT NULL,
	type VARCHAR(32) NOT NULL,
	destination VARCHAR(255) NOT NULL,
	data blob NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE KEY (uuid, destination),
	PRIMARY KEY (id)
);

-- incoming_raw_message persists incoming text based messages (email, sms)
-- in their raw form.
CREATE TABLE incoming_raw_message (
	id BIGINT UNSIGNED NOT NULL,
	type VARCHAR(32) NOT NULL,
	data blob NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);

-- renaming excomms_event to twilio_call_event so as to capture the fact that 
-- it merely represents call events from twilio.
RENAME TABLE excomms_event TO twilio_call_event;