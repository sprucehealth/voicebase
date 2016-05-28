CREATE TABLE auth.account ( 
	id                   bigint UNSIGNED NOT NULL,
	first_name           varchar(150),
	last_name            varchar(150),
	primary_account_email_id bigint UNSIGNED,
	primary_account_phone_id bigint UNSIGNED,
	password 			 varbinary(250) NOT NULL,
	status               varchar(100),
	created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified        	 timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX      idx_status (status),
	CONSTRAINT pk_account PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE auth.auth_token (
	token               varbinary(250) NOT NULL,
	account_id          bigint UNSIGNED NOT NULL,
	created             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires          	timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT fk_auth_token_account_id FOREIGN KEY (account_id) REFERENCES auth.account(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
	CONSTRAINT pk_auth_token PRIMARY KEY (token)
) engine=InnoDB;

CREATE TABLE auth.account_event (
	id                   bigint UNSIGNED NOT NULL,
	created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	account_id           bigint UNSIGNED NOT NULL,
	account_email_id     bigint UNSIGNED,
	account_phone_id     bigint UNSIGNED,
	event                text NOT NULL,
	CONSTRAINT pk_account_event PRIMARY KEY (id),
	CONSTRAINT fk_account_event_account_id FOREIGN KEY (account_id) REFERENCES auth.account(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;

CREATE TABLE auth.account_phone (
	id                   bigint UNSIGNED NOT NULL,
	account_id           bigint UNSIGNED NOT NULL,
	phone_number         varchar(50) NOT NULL,
	status               varchar(100) NOT NULL,
	verified             bool,
	created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX      idx_status (status),
	CONSTRAINT pk_account_phone PRIMARY KEY (id),
	CONSTRAINT fk_account_phone_account_id FOREIGN KEY (account_id) REFERENCES auth.account(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;

CREATE TABLE auth.account_email (
	id                   bigint UNSIGNED NOT NULL,
	account_id           bigint UNSIGNED NOT NULL,
	email                varchar(100) NOT NULL,
	status               varchar(100) NOT NULL,
	verified             bool NOT NULL DEFAULT false,
	created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	UNIQUE 	   (email),
	INDEX      idx_status (status),
	CONSTRAINT pk_account_email PRIMARY KEY (id),
	CONSTRAINT fk_account_email_account_id FOREIGN KEY (account_id) REFERENCES auth.account(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;

 ALTER TABLE auth.account ADD CONSTRAINT fk_account_primary_account_email_id_account_email_id FOREIGN KEY (primary_account_email_id) REFERENCES auth.account_email(id) ON DELETE NO ACTION ON UPDATE NO ACTION;
 ALTER TABLE auth.account ADD CONSTRAINT fk_account_primary_account_phone_id_account_phone_id FOREIGN KEY (primary_account_phone_id) REFERENCES auth.account_phone(id) ON DELETE NO ACTION ON UPDATE NO ACTION;
