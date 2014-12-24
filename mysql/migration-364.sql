ALTER TABLE doctor_queue ADD COLUMN short_description text NOT NULL;
ALTER TABLE doctor_queue ADD COLUMN patient_id INT UNSIGNED NOT NULL REFERENCES PATIENT(id);

ALTER TABLE unclaimed_case_queue ADD COLUMN patient_id INT UNSIGNED NOT NULL REFERENCES patient(id);
ALTER TABLE unclaimed_case_queue ADD COLUMN short_description text NOT NULL;

UPDATE unclaimed_case_queue as u
INNER JOIN patient_case pc ON pc.id = u.patient_case_id
SET u.patient_id = pc.patient_id, u.short_description = 'New visit';
