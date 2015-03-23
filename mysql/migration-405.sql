-- Admin site permissions for case views
INSERT INTO account_available_permission (name) VALUES ('case.view');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('case');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'case'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'case.%';

-- Add column to track what doctor was requested by this patient
ALTER TABLE patient_case
ADD COLUMN requested_doctor_id INT(10) UNSIGNED,
ADD CONSTRAINT fk_requested_doctor_doctor_id 
FOREIGN KEY(requested_doctor_id)
REFERENCES doctor(id);
