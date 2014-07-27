ALTER TABLE case_notification DROP KEY uid;
ALTER TABLE case_notification ADD UNIQUE KEY (patient_case_id, uid);
