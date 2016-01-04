CREATE TABLE blackbox.suite_run ( 
	id                   bigint UNSIGNED NOT NULL AUTO_INCREMENT,
	suite_name           varchar(250) NOT NULL,
	status               varchar(100) NOT NULL,
	tests_passed         int UNSIGNED NOT NULL,
	tests_failed         int UNSIGNED NOT NULL,
	start                timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	finish          	 timestamp,
	INDEX      idx_status (status),
	CONSTRAINT pk_suite_run PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE blackbox.suite_test_run (
	id                  bigint UNSIGNED NOT NULL AUTO_INCREMENT,
	suite_run_id        bigint UNSIGNED NOT NULL,
	test_name           varchar(250) NOT NULL,
	status              varchar(100) NOT NULL,
	message             text,
	start               timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	finish          	timestamp,
	CONSTRAINT fk_suite_test_run_suite_run_id FOREIGN KEY (suite_run_id) REFERENCES blackbox.suite_run(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
	CONSTRAINT pk_suite_test_run PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE blackbox.profile ( 
	id                   bigint UNSIGNED NOT NULL AUTO_INCREMENT,
	profile_key          varchar(250) NOT NULL,
	result_ms            float UNSIGNED NOT NULL,
	created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT pk_blackbox PRIMARY KEY (id)
) engine=InnoDB;