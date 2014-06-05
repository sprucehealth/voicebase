CREATE TABLE resource_guide_section (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	ordinal INT NOT NULL,
	title VARCHAR(256) NOT NULL,
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE resource_guide (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	section_id INT UNSIGNED NOT NULL,
	ordinal INT NOT NULL,
	title VARCHAR(256) NOT NULL,
	photo_url VARCHAR(256) NOT NULL,
	layout BLOB NOT NULL,
	creation_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	FOREIGN KEY (section_id) REFERENCES resource_guide_section (id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;
