-- CREATE the tags table
CREATE TABLE tag (
  id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
  tag_text VARCHAR(255),
  PRIMARY KEY (id),
  UNIQUE KEY tag_text (tag_text)
);

-- CREATE the memberships table
CREATE TABLE tag_membership (
  tag_id INT(10) UNSIGNED NOT NULL,
  case_id INT(10) UNSIGNED,
  trigger_time TIMESTAMP,
  hidden BOOLEAN NOT NULL,
  PRIMARY KEY (tag_id, case_id),
  FOREIGN KEY (tag_id) REFERENCES tag (id) ON DELETE CASCADE,
  FOREIGN KEY (case_id) REFERENCES patient_case (id) ON DELETE CASCADE
);