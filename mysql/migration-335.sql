ALTER TABLE form_doctor_interest ADD COLUMN source VARCHAR(64);
ALTER TABLE form_notify_me ADD COLUMN source VARCHAR(64);

UPDATE form_doctor_interest SET source = 'home';
UPDATE form_notify_me SET source = 'home';

ALTER TABLE form_doctor_interest CHANGE COLUMN source source VARCHAR(64) NOT NULL;
ALTER TABLE form_notify_me CHANGE COLUMN source source VARCHAR(64) NOT NULL;
