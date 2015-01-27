CREATE TABLE common_diagnosis_set (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	pathway_id INT UNSIGNED NOT NULL,
	title varchar(600) NOT NULL,
	FOREIGN KEY (pathway_id) REFERENCES clinical_pathway(id),
	PRIMARY KEY (id)
) CHARACTER SET UTF8;

CREATE TABLE common_diagnosis_set_item (
 	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
 	diagnosis_code_id VARCHAR(32) NOT NULL,
 	active tinyint(1) NOT NULL DEFAULT 1,
 	common_diagnosis_set_id INT UNSIGNED NOT NULL,
 	FOREIGN KEY (common_diagnosis_set_id) REFERENCES common_diagnosis_set(id),
 	KEY (common_diagnosis_set_id, active),
 	PRIMARY KEY (id)
) CHARACTER SET UTF8;