-- Admin site permissions for scheduled messages
INSERT INTO account_available_permission (name) VALUES ('sched_msgs.view'), ('sched_msgs.edit');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('sched_msgs');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'sched_msgs'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'sched_msgs.%';
