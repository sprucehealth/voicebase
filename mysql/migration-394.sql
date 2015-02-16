INSERT INTO account_available_permission (name)
	VALUES ('resource_guides.view'), ('resource_guides.edit'), ('rx_guides.view'), ('rx_guides.edit');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;
INSERT INTO account_group (name) VALUES ('resource_guides'), ('rx_guides');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'resource_guides'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'resource_guides.%';
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'rx_guides'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'rx_guides.%';

ALTER TABLE resource_guide ADD COLUMN active BOOL NOT NULL DEFAULT 0;
UPDATE resource_guide SET active = 1;
ALTER TABLE resource_guide ADD KEY resource_guide_active_ordinal (active, ordinal);
