-- Admin site permissions for financial views
INSERT INTO account_available_permission (name) VALUES ('financial.view');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('financial');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'financial'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'financial.%';

ALTER TABLE patient_receipt ADD INDEX (creation_timestamp);
ALTER TABLE doctor_transaction ADD INDEX (created);
