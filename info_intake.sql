use info_intake_db;

CREATE TABLE IF NOT EXISTS app_text (
	id int(11) NOT NULL AUTO_INCREMENT,
	comment varchar(600),
	app_text_tag varchar(250) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY (app_text_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS localized_text (
	id int(11) NOT NULL AUTO_INCREMENT,
	language varchar(5) NOT NULL,
	ltext varchar(600) NOT NULL,
	app_text_id int(11) NOT NULL,
	FOREIGN KEY (app_text_id) REFERENCES app_text(id) ON DELETE CASCADE,
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question_type (
	id int(11) NOT NULL AUTO_INCREMENT,
	qType varchar(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

// unique constraints
// date associated with each answer
// there should be a way to specify previously selected answers.
CREATE TABLE IF NOT EXISTS question (
	id int(11) NOT NULL AUTO_INCREMENT,
	qType_id int(11) NOT NULL,
	qtext_app_text_id int(11) NOT NULL,
	subtext_app_text_id int(11) NOT NULL,
	section_id int(11) NOT NULL,
	question_tag varchar(250) NOT NULL,
	FOREIGN KEY (qType_id) REFERENCES question_type(id),
	FOREIGN KEY (subtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (qtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (section_id) REFERENCES section(id),
	PRIMARY KEY (id),
	UNIQUE KEY (question_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS outcome_type (
	id int(11) NOT NULL AUTO_INCREMENT,
	otype varchar(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS potential_outcome (
	id int(11) NOT NULL AUTO_INCREMENT,
	question_id int(11) NOT NULL,
	outcome_localized_text int(11) NOT NULL,
	otype_id int(11),
	potential_outcome_tag varchar(250) NOT NULL,
	FOREIGN KEY (otype_id) REFERENCES outcome_type(id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (outcome_localized_text) REFERENCES app_text(id),
	PRIMARY KEY (id),
	UNIQUE KEY (potential_outcome_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS section (
	id int(11) NOT NULL AUTO_INCREMENT,
	section_title_app_text_id int(11) NOT NULL,
	comment varchar(600) NOT NULL,
	treatment_id int(11) NOT NULL,
	section_tag varchar(250) NOT NULL,
	FOREIGN KEY (section_title_app_text_id) REFERENCES app_text(id),
	PRIMARY KEY (id),	
	UNIQUE KEY (section_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS treatment (
	id int(11) NOT NULL AUTO_INCREMENT,
	comment varchar(600) NOT NULL,
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS patient_info_intake (
	id int(11) NOT NULL AUTO_INCREMENT,
	treatment_id int(11) NOT NULL,
	question_id int(11) NOT NULL,
	potential_outcome_id int(11) NOT NULL,
	outcome_text 
	PRIMARY KEY (id),
	FOREIGN KEY (question_id) REFERENCES question(id),
) CHARACTER SET UTF8;


