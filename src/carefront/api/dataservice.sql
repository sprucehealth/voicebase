use carefront_db;

CREATE TABLE IF NOT EXISTS account (
	id int unsigned NOT NULL AUTO_INCREMENT,
	email varchar(250),
	password varbinary(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS auth_token (
	token varbinary(250),
	account_id int unsigned not null,
	created timestamp NOT NULL,
	expires timestamp NOT NULL,
	PRIMARY KEY (token),
	FOREIGN KEY (account_id) REFERENCES account(id) ON DELETE CASCADE
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS patient (
	id int unsigned not null AUTO_INCREMENT,
	first_name varchar(500) not null,
	last_name varchar(500) not null,
	dob timestamp not null,
	gender varchar(500) not null,
	zip_code varchar(500) not null,
	status varchar (500) not null,
	PRIMARY KEY(id)
) CHARACTER SET utf8;

create table if not exists patient_visit (
	id int unsigned not null AUTO_INCREMENT,
	patient_id int unsigned not null,
	creation_date timestamp not null default current_timestamp,
	opened_date timestamp,
	closed_date timestamp,	
	treatment_id int unsigned not null,
	status varchar(100) not null,
	FOREIGN KEY  (patient_id) REFERENCES patient(id),
	FOREIGN KEY  (treatment_id) REFERENCES treatment(id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;