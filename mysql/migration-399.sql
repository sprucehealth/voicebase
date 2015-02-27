-- Admin site permissions for pathways
INSERT INTO account_available_permission (name) VALUES ('ftp.view'), ('ftp.edit');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;
INSERT INTO account_group (name) VALUES ('ftp');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'ftp'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'ftp.%';

-- Create a lifecycle attribute for FTPs
ALTER TABLE dr_favorite_treatment_plan ADD COLUMN lifecycle varchar(20);
UPDATE dr_favorite_treatment_plan SET lifecycle = 'ACTIVE';
ALTER TABLE dr_favorite_treatment_plan MODIFY lifecycle varchar(20) NOT NULL;