use carefront_db;

CREATE TABLE IF NOT EXISTS languages_supported (
	id int unsigned NOT NULL AUTO_INCREMENT,
	language varcha(10) NOT NULL,
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
	UNIQUE KEY (language_id, app_text_id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question_type (
	id int unsigned NOT NULL AUTO_INCREMENT,
	qtype varchar(250),
	UNIQUE KEY (qtype),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS question (
	id int unsigned NOT NULL AUTO_INCREMENT,
	qtype_id int unsigned NOT NULL,
	qtext_app_text_id int unsigned,	
	qtext_short_text_id int unsigned NOT NULL,
	subtext_app_text_id int unsigned,
	question_tag varchar(250) NOT NULL,
	parent_question_id int unsigned,
	required bool not null,
	FOREIGN KEY (qtype_id) REFERENCES question_type(id),
	FOREIGN KEY (subtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (qtext_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (qtext_short_text_id) REFERENCES app_text(id),
	FOREIGN KEY (parent_question_id) REFERENCES question(id),
	PRIMARY KEY (id),
	UNIQUE KEY (question_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS answer_type (
	id int unsigned NOT NULL AUTO_INCREMENT,
	atype varchar(250),
	UNIQUE KEY (atype),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS potential_answer (
	id int unsigned NOT NULL AUTO_INCREMENT,
	question_id int unsigned NOT NULL,
	answer_localized_text int unsigned NOT NULL,
	summary_localized_text int unisgned,
	atype_id int unsigned NOT NULL,
	potential_answer_tag varchar(250) NOT NULL,
	ordering int unsigned NOT NULL,
	FOREIGN KEY (atype_id) REFERENCES answer_type(id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (answer_localized_text) REFERENCES app_text(id),
	FOREIGN KEY (summary_localized_text) REFERENCES summary_localized_text(id),
	PRIMARY KEY (id),
	UNIQUE KEY (potential_answer_tag),
	UNIQUE KEY (question_id, otype_id),
	UNIQUE KEY (question_id, ordering)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS section (
	id int unsigned NOT NULL AUTO_INCREMENT,
	section_title_app_text_id int unsigned NOT NULL,
	comment varchar(600) NOT NULL,
	health_condition_id int unsigned, 
	section_tag varchar(250) NOT NULL,
	FOREIGN KEY (section_title_app_text_id) REFERENCES app_text(id),
	FOREIGN KEY (health_condition_id) REFERENCES health_condition(id),
	PRIMARY KEY (id),	
	UNIQUE KEY (section_tag)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS screen_type (
	id int unsigned NOT NULL AUTO_INCREMENT,
	screen_type_tag varchar(100) NOT NULL,
	UNIQUE KEY (screen_type_tag),
	PRIMARY KEY(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS health_condition (
	id int unsigned NOT NULL AUTO_INCREMENT,
	comment varchar(600) NOT NULL,
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS patient_info_intake (
	id int unsigned NOT NULL AUTO_INCREMENT,
	patient_id int unsigned not null,
	case_id int unsigned,
	question_id int unsigned NOT NULL,
	section_id int unsigned NOT NULL,
	potential_answer_id int unsigned NOT NULL,
	answer_text varchar(600),
	client_layout_version_id int unsigned NOT NULL,
	answered_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	parent_info_intake_id int unsigned,
	status varchar(100) NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY (question_id) REFERENCES question(id),
	FOREIGN KEY (client_layout_version_id) REFERENCES patient_layout_version(id),
	FOREIGN KEY (patient_id) REFERENCES patient(id),
	FOREIGN KEY (section_id) REFERENCES section(id),
	FOREIGN KEY (parent_info_intake_id) REFERENCES patient_info_intake(id)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	syntax_version int unsigned NOT NULL,
	health_condition_id int unsigned NOT NULL,
	comment varchar(600),
	status varchar(250) NOT NULL, 
	creation_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date timestamp NOT NULL,
	FOREIGN KEY (health_condition_id) REFERENCES health_condition(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	PRIMARY KEY (id),
	UNIQUE KEY (object_storage_id, syntax_version, health_condition_id, status)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS patient_layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	language_id int unsigned NOT NULL,
	health_condition_id int unsigned NOT NULL,
	layout_version_id int unsigned NOT NULL,
	status varchar(250) NOT NULL, 
	creation_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date timestamp NOT NULL,
	FOREIGN KEY (layout_version_id) REFERENCES layout_version(id),
	FOREIGN KEY (language_id) REFERENCES languages_supported(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	FOREIGN KEY (health_condition_id) REFERENCES health_condition(id),
	PRIMARY KEY(id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS dr_layout_version (
	id int unsigned NOT NULL AUTO_INCREMENT,
	object_storage_id int unsigned NOT NULL,
	layout_version_id int unsigned NOT NULL,
	status varchar(250) NOT NULL,
	creation_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date timestamp NOT NULL,
	FOREIGN KEY (layout_version_id) REFERENCES layout_version(id),
	FOREIGN KEY (object_storage_id) REFERENCES object_storage(id),
	PRIMARY KEY(id)
) CHARACTER SET UTF8;


CREATE TABLE IF NOT EXISTS object_storage (
	id int unsigned NOT NULL AUTO_INCREMENT,
	region_id int unsigned NOT NULL,
	bucket varchar(100) NOT NULL,
	storage_key varchar(100) NOT NULL,
	status varchar(100) NOT NULL,
	creation_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date timestamp NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY (region_id) REFERENCES region(id),
	UNIQUE KEY (region_id, storage_key, bucket, status)
) CHARACTER SET UTF8;

CREATE TABLE IF NOT EXISTS region (
	id int unsigned NOT NULL AUTO_INCREMENT,
	region_tag varchar(100) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY (region_tag)
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
	tips_title_text_id int unsigned NOT NULL,
	tips_subtext_text_id int unsigned NOT NULL,
	comment varchar(500),
	UNIQUE KEY (tips_section_tag),
	FOREIGN KEY (tips_title_text_id) REFERENCES app_text(id),
	FOREIGN KEY (tips_subtext_text_id) REFERENCES app_text(id),
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

