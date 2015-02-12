ALTER TABLE doctor_queue ADD COLUMN tags varchar(128) NOT NULL;
ALTER TABLE unclaimed_case_queue ADD COLUMN tags varchar(128) NOT NULL;