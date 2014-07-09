CREATE TABLE account_phone (
  id int UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id INT UNSIGNED NOT NULL,
  phone VARCHAR(64) NOT NULL,
  phone_type VARCHAR(32) NOT NULL,
  status varchar(32) NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (account_id) REFERENCES account (id)
) CHARACTER SET utf8;

INSERT INTO account_phone (account_id, phone, phone_type, status)
	SELECT doctor.account_id, phone, phone_type, 'ACTIVE'
	FROM doctor_phone
	INNER JOIN doctor ON doctor.id = doctor_id;

INSERT INTO account_phone (account_id, phone, phone_type, status)
	SELECT patient.account_id, phone, phone_type, patient_phone.status
	FROM patient_phone
	INNER JOIN patient ON patient.id = patient_id;

-- The doctor_phone table was using MAIN for the type. Unify on one standard set of
-- types and assume the doctor is using a Cell number
UPDATE account_phone SET phone_type = 'Cell' WHERE phone_type = 'MAIN';

DROP TABLE doctor_phone;
DROP TABLE patient_phone;

-- Seems Maine is using 'US' when the rest are using 'USA'
UPDATE state SET country = 'USA' where country = 'US';

CREATE TABLE doctor_attribute (
  id int UNSIGNED NOT NULL AUTO_INCREMENT,
  doctor_id INT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  value VARCHAR(1024) NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (doctor_id) REFERENCES doctor (id),
  UNIQUE KEY (doctor_id, name)
) CHARACTER SET utf8;

CREATE TABLE doctor_medical_license (
  id int UNSIGNED NOT NULL AUTO_INCREMENT,
  doctor_id INT UNSIGNED NOT NULL,
  state CHAR(2) NOT NULL,
  license_number VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  PRIMARY KEY (id),
  FOREIGN KEY (doctor_id) REFERENCES doctor (id),
  UNIQUE KEY (doctor_id, state)
) CHARACTER SET utf8;

CREATE TABLE bank_account (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id INT UNSIGNED NOT NULL,
  stripe_recipient_id VARCHAR(128) NOT NULL,
  default_account BOOL NOT NULL,
  verified BOOL NOT NULL DEFAULT false,
  verify_amount_1 INT,
  verify_amount_2 INT,
  verify_transfer1_id VARCHAR(128),
  verify_transfer2_id VARCHAR(128),
  verify_expires TIMESTAMP,
  creation_date TIMESTAMP NOT NULL DEFAULT current_timestamp,
  PRIMARY KEY (id),
  FOREIGN KEY (account_id) REFERENCES account (id)
) CHARACTER SET utf8;
