CREATE TABLE blocked_accounts (
	account_id VARCHAR(64) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (account_id)
);