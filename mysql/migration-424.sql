-- Admin site permissions for merketing interaction
INSERT INTO account_available_permission (name) VALUES ('marketing.view'), ('marketing.edit');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('marketing');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'marketing'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'marketing.%';

-- Create the routes table
-- This table should stay relativley small so the nullable indexes shouldn't be a problem
CREATE TABLE promotion_referral_route (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  promotion_code_id INT UNSIGNED NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT current_timestamp,
  modified TIMESTAMP NOT NULL DEFAULT current_timestamp ON UPDATE CURRENT_TIMESTAMP,
  priority INT UNSIGNED NOT NULL,
  lifecycle VARCHAR(25) NOT NULL DEFAULT 'ACTIVE',
  gender VARCHAR(1),
  age_lower INT UNSIGNED,
  age_upper INT UNSIGNED,
  state VARCHAR(2),
  pharmacy VARCHAR(255),
  PRIMARY KEY (id),
  INDEX promotion_referral_route_priority_idx (priority),
  INDEX promotion_referral_route_gender_idx (gender),
  INDEX promotion_referral_route_age_lower_idx (age_lower),
  INDEX promotion_referral_route_age_upper_idx (age_upper),
  INDEX promotion_referral_route_state_idx (state),
  INDEX promotion_referral_route_pharmacy_idx (pharmacy),
  CONSTRAINT promotion_referral_route_promotion_code FOREIGN KEY (promotion_code_id) REFERENCES promotion_code (id)
);

-- Move our single referral template into the default state
UPDATE referral_program_template SET status = 'Default' WHERE status = 'Active';

-- Map Referral Program Templates to the Promotion
ALTER TABLE referral_program_template
  ADD COLUMN promotion_code_id INT UNSIGNED,
  ADD CONSTRAINT referral_program_template_promotion_code FOREIGN KEY (promotion_code_id) REFERENCES promotion_code (id);

-- Map Referral Programs to the originating route
ALTER TABLE referral_program
  ADD COLUMN promotion_referral_route_id INT UNSIGNED,
  ADD CONSTRAINT referral_program_promotion_referral_route FOREIGN KEY (promotion_referral_route_id) REFERENCES promotion_referral_route (id);