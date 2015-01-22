ALTER TABLE form_notify_me ADD COLUMN unique_key varchar(128);
ALTER TABLE form_notify_me ADD UNIQUE KEY (unique_key);
ALTER TABLE treatment_plan ADD COLUMN patient_viewed tinyint(1) NOT NULL DEFAULT 0;
