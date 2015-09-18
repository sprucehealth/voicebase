-- Add table for attribution information
CREATE TABLE attribution_data (
  id MEDIUMINT NOT NULL AUTO_INCREMENT,
  account_id INT UNSIGNED,
  device_id VARCHAR(128),
  json_data BLOB NOT NULL,
  creation_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  INDEX account_id_idx (account_id),
  INDEX device_id_idx (device_id),
  CONSTRAINT attribution_data_account_id FOREIGN KEY (account_id) REFERENCES account(id));