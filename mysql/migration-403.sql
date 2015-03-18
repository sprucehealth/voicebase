-- Modify source_id to be more specific for now 
ALTER TABLE patient_alerts CHANGE source_id question_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_alerts ADD FOREIGN KEY (question_id) REFERENCES question(id);

-- Get rid of source given that only the visit intake can produce alerts for now
ALTER TABLE patient_alerts DROP COLUMN source;

-- Add more context to the alerts table to tie the alerts to the visit level
ALTER TABLE patient_alerts ADD COLUMN patient_visit_id INT UNSIGNED;

-- Update all alerts that are currently ACTIVE to tie them to the latest visit of the patient
UPDATE patient_alerts
	SET patient_visit_id = (SELECT id FROM patient_visit WHERE patient_id = patient_alerts.patient_id ORDER BY id DESC LIMIT 1)
	WHERE status = 'ACTIVE';

-- Update all alerts that are currently INACTIVE to tie them to the oldest visit of the patient
UPDATE patient_alerts
	SET patient_visit_id = (SELECT id FROM patient_visit WHERE patient_id = patient_alerts.patient_id ORDER BY id LIMIT 1)
	WHERE status = 'INACTIVE';

ALTER TABLE patient_alerts DROP COLUMN status;
ALTER TABLE patient_alerts ADD FOREIGN KEY (patient_visit_id) REFERENCES patient_visit(id);
ALTER TABLE patient_alerts MODIFY COLUMN patient_visit_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_alerts DROP FOREIGN KEY patient_alerts_ibfk_1;
ALTER TABLE patient_alerts DROP COLUMN patient_id;
