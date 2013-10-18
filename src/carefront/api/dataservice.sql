use carefront_db;

CREATE TABLE IF NOT EXISTS Account (
	id int(11) NOT NULL AUTO_INCREMENT,
	email varchar(250),
	password varbinary(250),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS Token (
	token varbinary(250),
	account_id int(11),
	created timestamp NOT NULL,
	expires timestamp NOT NULL,
	PRIMARY KEY (token),
	FOREIGN KEY (account_id) REFERENCES Account(id) ON DELETE CASCADE
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS CaseImage (
	id int(11) NOT NULL AUTO_INCREMENT,
	case_id int(11),
	photoType ENUM('FACE_MIDDLE', 'FACE_RIGHT', 'FACE_LEFT', 'CHEST', 'BACK'),
	status ENUM('PENDING_UPLOAD', 'PENDING_APPROVAL', 'REJECTED', 'APPROVED'),
	uploadDate timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
) CHARACTER SET utf8;

ALTER TABLE CaseImage ADD INDEX caseId_status_index (case_id, status);
ALTER TABLE CaseImage ADD INDEX caseIdIndex (case_id);