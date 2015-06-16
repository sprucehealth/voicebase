CREATE TABLE patient_case_message_read (
	"message_id" BIGINT(20) UNSIGNED NOT NULL,
	"person_id" BIGINT(20) UNSIGNED NOT NULL,
	"timestamp" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY ("message_id", "person_id")
);

ALTER TABLE patient_case_message_participant DROP COLUMN unread;
ALTER TABLE patient_case_message_participant DROP COLUMN last_read_tstamp;
