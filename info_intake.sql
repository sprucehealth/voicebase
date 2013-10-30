use info_intake_db;

CREATE TABLE IF NOT EXISTS app_text (
	id unsigned int NOT NULL AUTO_INCREMENT,
	comment varchar(600),
	app_text_tag varchar(250) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY (app_text_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS localized_text (
	id unsigned int NOT NULL AUTO_INCREMENT,
	language varchar(5) NOT NULL,
	ltext varchar(600) NOT NULL,
	app_text_id int unsigned NOT NULL,
	FOREIGN KEY (app_text_id) REFERENCES app_text(id) ON DELETE CASCADE,
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question_type (
	id unsigned int NOT NULL AUTO_INCREMENT,
	qType varchar(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question (
	id unsigned int NOT NULL AUTO_INCREMENT,
	qType_id int unsigned NOT NULL,
	qtext_app_text_id int unsigned NOT NULL,
	subtext_app_text_id int unsigned NOT NULL,
	section_id int unsigned NOT NULL,
	question_tag varchar(250) NOT NULL,
	FOREIGN KEY (qType_id) REFERENCES question_type(id),
	FOREIGN KEY (subtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (qtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (section_id) REFERENCES section(id),
	PRIMARY KEY (id),
	UNIQUE KEY (question_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS outcome_type (
	id int unsigned NOT NULL AUTO_INCREMENT,
	otype varchar(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS potential_outcome (
	id int unsigned NOT NULL AUTO_INCREMENT,
	question_id int unsigned NOT NULL,
	outcome_localized_text int unsigned NOT NULL,
	otype_id int unsigned,
	potential_outcome_tag varchar(250) NOT NULL,
	FOREIGN KEY (otype_id) REFERENCES outcome_type(id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (outcome_localized_text) REFERENCES app_text(id),
	PRIMARY KEY (id),
	UNIQUE KEY (potential_outcome_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS section (
	id int unsigned NOT NULL AUTO_INCREMENT,
	section_title_app_text_id int unsigned NOT NULL,
	comment varchar(600) NOT NULL,
	treatment_id int unsigned NOT NULL,
	section_tag varchar(250) NOT NULL,
	FOREIGN KEY (section_title_app_text_id) REFERENCES app_text(id),
	PRIMARY KEY (id),	
	UNIQUE KEY (section_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS treatment (
	id int unsigned NOT NULL AUTO_INCREMENT,
	comment varchar(600) NOT NULL,
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS patient_info_intake (
	id int unsigned NOT NULL AUTO_INCREMENT,
	case_id int unsigned,
	question_id int unsigned NOT NULL,
	potential_outcome_id int unsigned NOT NULL,
	outcome_text varchar(600),
	layout_version_id int unsigned NOT NULL,
	answered_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	status varchar(100) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (layout_version_id) REFERENCES layout_version(id),
	FOREIGN KEY (case_id) REFERENCES case(id),
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	url varchar(250) NOT NULL,
	treatment_id int unsigned NOT NULL,
	comment varchar(600),
	status varchar(250) NOT NULL, 
	FOREIGN KEY (treatment_id) REFERENCES treatment(id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;


