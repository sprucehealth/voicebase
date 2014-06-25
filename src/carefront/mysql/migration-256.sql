
INSERT INTO patient_case_message (body, person_id, tstamp, patient_case_id)
  SELECT diagnosis_summary.summary, person.id, diagnosis_summary.creation_date, patient_case.id
  FROM diagnosis_summary
  INNER JOIN role_type ON role_type.role_type_tag = 'DOCTOR'
  INNER JOIN person ON person.role_id = doctor_id AND person.role_type_id = role_type.id
  INNER JOIN treatment_plan ON treatment_plan.id = diagnosis_summary.treatment_plan_id
  INNER JOIN patient_case ON patient_case.patient_id = treatment_plan.patient_id;

REPLACE INTO patient_case_message_participant (patient_case_id, person_id)
  SELECT DISTINCT patient_case_id, person_id FROM patient_case_message;

DROP TABLE diagnosis_summary;

CREATE TABLE doctor_saved_case_message (
  doctor_id INT UNSIGNED NOT NULL,
  message TEXT NOT NULL,
  creation_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  modified_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (doctor_id) REFERENCES doctor (id),
  PRIMARY KEY (doctor_id)
);
