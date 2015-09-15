CREATE TABLE practice_model(
  doctor_id INT UNSIGNED NOT NULL,
  spruce_pc BOOL NOT NULL DEFAULT false,
  practice_extension BOOL NOT NULL DEFAULT false,
  PRIMARY KEY (doctor_id),
  CONSTRAINT practice_model_doctor_id FOREIGN KEY (doctor_id) REFERENCES doctor(id));

INSERT INTO practice_model (doctor_id, spruce_pc)
  SELECT id, true FROM doctor;