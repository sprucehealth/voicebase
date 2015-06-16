-- Create the table for tracking case notes
CREATE TABLE patient_case_note (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  case_id INT UNSIGNED NOT NULL,
  author_doctor_id INT UNSIGNED NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT current_timestamp,
  modified TIMESTAMP NOT NULL DEFAULT current_timestamp ON UPDATE CURRENT_TIMESTAMP,
  note_text TEXT CHARACTER SET utf8mb4 NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT patient_case_note_patient_case FOREIGN KEY (case_id) REFERENCES patient_case (id),
  CONSTRAINT patient_case_note_doctor FOREIGN KEY (author_doctor_id) REFERENCES doctor (id)
);