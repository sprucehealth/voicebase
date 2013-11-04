use info_intake_db;

CREATE TABLE IF NOT EXISTS languages_supported (
	id int unsigned NOT NULL AUTO_INCREMENT,
	language varhcar(10) NOT NULL,
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS app_text (
	id int unsigned NOT NULL AUTO_INCREMENT,
	comment varchar(600),
	app_text_tag varchar(250) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY (app_text_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS localized_text (
	id int unsigned NOT NULL AUTO_INCREMENT,
	language_id int unsigned NOT NULL,
	ltext varchar(600) NOT NULL,
	app_text_id int unsigned NOT NULL,
	FOREIGN KEY (app_text_id) REFERENCES app_text(id) ON DELETE CASCADE,
	FOREIGN KEY (language_id) REFERENCES languages_supported(id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question_type (
	id int unsigned NOT NULL AUTO_INCREMENT,
	qtype varchar(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question (
	id int unsigned NOT NULL AUTO_INCREMENT,
	qtype_id int unsigned NOT NULL,
	qtext_app_text_id int unsigned NOT NULL,
	subtext_app_text_id int unsigned NOT NULL,
	question_tag varchar(250) NOT NULL,
	FOREIGN KEY (qtype_id) REFERENCES question_type(id),
	FOREIGN KEY (subtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (qtext_app_text_id) REFERENCES app_text(id),
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
	otype_id int unsigned NOT NULL,
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
	FOREIGN KEY (treatment_id) REFERENCES treatment(id),
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
	section_id int unsigned NOT NULL,
	potential_outcome_id int unsigned NOT NULL,
	outcome_text varchar(600),
	client_layout_version_id int unsigned NOT NULL,
	answered_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	status varchar(100) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (client_layout_version_id) REFERENCES client_layout_version(id),
	FOREIGN KEY (case_id) REFERENCES case(id),
	FOREIGN KEY (section_id) REFERENCES section(id),
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	syntax_version int unsigned NOT NULL,
	treatment_id int unsigned NOT NULL,
	comment varchar(600),
	status varchar(250) NOT NULL, 
	FOREIGN KEY (treatment_id) REFERENCES treatment(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS patient_layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	language_id int unsigned NOT NULL,
	layout_version_id int unsigned NOT NULL,
	status varchar(250) NOT NULL, 
	FOREIGN KEY (layout_version_id) REFERENCES layout_version(id),
	FOREIGN KEY (language_id) REFERENCES languages_supported(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	PRIMARY KEY(id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS dr_layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	layout_version_id int unsigned NOT NULL,
	status varchar(250) NOT NULL,
	FOREIGN KEY (layout_version_id) REFERENCES layout_version(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS client_hardcoded_screen (
	id int unsigned NOT NULL AUTO_INCREMENT,
	client_hardcoded_screen_tag varchar(100) NOT NULL,
	app_version varchar(10) NOT NULL,
	UNIQUE KEY (client_hardcoded_screen_tag),
	UNIQUE KEY (appVersion),
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS object_storage (
	id int unsigned NOT NULL AUTO_INCREMENT,
	region varchar(100) NOT NULL,
	bucket varchar(100) NOT NULL,
	key varchar(100) NOT NULL,
	status varchar(100) NOT NULL,
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS tips (
	id int unsigned NOT NULL AUTO_INCREMENT,
	tips_text_id int unsigned NOT NULL,
	tips_tag varchar(100) NOT NULL,
	FOREIGN KEY (tips_text_id) REFERENCES app_text(id),
	UNIQUE KEY(tips_tag),
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS tips_section (
	id int unsigned NOT NULL AUTO_INCREMENT,
	tips_section_tag varchar(100) NOT NULL,
	comment varchar(500),
	potential_outcome_id int unsigned NOT NULL,
	UNIQUE KEY (tips_section_tag),
	FOREIGN KEY (potential_outcome_id) REFERENCES potential_outcome(id),
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS photo_tips (
	id int unsigned NOT NULL AUTO_INCREMENT,
	photo_tips_tag varchar(100) NOT NULL,
	photo_url_id int unsigned NOT NULL,
	UNIQUE KEY (photo_tips_tag),
	FOREIGN KEY (photo_url_id) REFERENCES object_storage(id),
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

