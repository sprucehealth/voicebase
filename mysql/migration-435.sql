  -- Admin site permissions for account interaction
INSERT INTO account_available_permission (name) VALUES ('account.edit'), ('account.view');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('account');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'account'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'account.%';