-- Link messages to cases
ALTER TABLE conversation_message ADD COLUMN patient_case_id INT UNSIGNED NOT NULL;

UPDATE conversation_message AS m
    INNER JOIN (
        SELECT patient_case.id AS case_id, conversation_id
        FROM conversation_participant
        INNER JOIN person ON person.id = person_id
        INNER JOIN role_type ON role_type.id = role_type_id
        INNER JOIN patient_case ON patient_id = role_id
        WHERE role_type_tag = 'PATIENT'
    ) AS p ON p.conversation_id = m.conversation_id
    SET patient_case_id = case_id;

ALTER TABLE conversation_message ADD FOREIGN KEY (patient_case_id) REFERENCES patient_case (id);
ALTER TABLE conversation_message ADD INDEX case_tstamp (patient_case_id, tstamp);

-- Unlink the conversation table and remove now unused tables

ALTER TABLE conversation_message DROP FOREIGN KEY conversation_message_ibfk_1;
ALTER TABLE conversation_message DROP INDEX conversation_id;
ALTER TABLE conversation_message DROP COLUMN conversation_id;

DROP TABLE conversation_participant;
DROP TABLE conversation;
DROP TABLE conversation_topic;

-- Rename the tables to reflect the new hierarchy to avoid confusion in the future if there's other message tables (e.g. Q&A)

ALTER TABLE conversation_message RENAME TO patient_case_message;
ALTER TABLE conversation_message_attachment RENAME TO patient_case_message_attachment;

-- Crete new participants table

CREATE TABLE patient_case_message_participant (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	patient_case_id INT UNSIGNED NOT NULL,
	person_id BIGINT UNSIGNED NOT NULL,
	unread BOOL NOT NULL DEFAULT 0,
	last_read_tstamp TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6),
	FOREIGN KEY (patient_case_id) REFERENCES patient_case (id),
	FOREIGN KEY (person_id) REFERENCES person (id),
	UNIQUE KEY (patient_case_id, person_id),
	PRIMARY KEY (id)
);

INSERT INTO patient_case_message_participant (patient_case_id, person_id)
	SELECT DISTINCT patient_case_id, person_id FROM patient_case_message;
