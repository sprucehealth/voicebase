ALTER TABLE tag_membership
  ADD COLUMN created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD INDEX tag_membership_created_idx (created);

ALTER TABLE tag
  ADD COLUMN common BOOLEAN DEFAULT false,
  ADD INDEX tag_common_idx(common);

  -- Admin site permissions for care coordinator interaction
INSERT INTO account_available_permission (name) VALUES ('care_coordinator.view'), ('care_coordinator.edit');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('care_coordinator');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'care_coordinator'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'care_coordinator.%';

CREATE TABLE tag_saved_search (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  title VARCHAR(50) NOT NULL,
  query TEXT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY tag_saved_search_title (title)
);