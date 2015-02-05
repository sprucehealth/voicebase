-- Migrate our new permissions
INSERT INTO account_available_permission (name) VALUES ('stp.view'), ('stp.edit');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;