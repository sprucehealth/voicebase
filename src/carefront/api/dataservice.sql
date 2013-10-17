use carefront_db;

CREATE TABLE IF NOT EXISTS Account (
	id int(11) NOT NULL AUTO_INCREMENT,
	email varchar(250),
	password varbinary(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS Token (
	token varbinary(250),
	account_id int(11),
	created timestamp NOT NULL,
	expires timestamp NOT NULL,
	PRIMARY KEY (token),
	FOREIGN KEY (account_id) REFERENCES Account(id) ON DELETE CASCADE
) CHARACTER SET utf8;
