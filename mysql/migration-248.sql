CREATE TABLE temp_auth_token (
	token VARCHAR(128) NOT NULL,
	expires TIMESTAMP NOT NULL,
	purpose VARCHAR(32) NOT NULL,
	account_id INT UNSIGNED NOT NULL,
	KEY (expires),
	FOREIGN KEY (account_id) REFERENCES account (id),
	PRIMARY KEY (purpose, token)
) CHARACTER SET utf8;
