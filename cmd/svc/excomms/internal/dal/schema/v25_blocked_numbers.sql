CREATE TABLE blocked_number (
	blocked_phone_number VARCHAR(16) NOT NULL,
  provisioned_phone_number VARCHAR(16) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (provisioned_phone_number, blocked_phone_number)
);
