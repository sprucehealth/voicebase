CREATE TABLE drug_description (
	drug_name_strength VARCHAR(250) NOT NULL,
	json BLOB NOT NULL,
	created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (drug_name_strength)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
