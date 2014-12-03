CREATE TABLE doctor_patient_case_feed (
    doctor_id INT UNSIGNED NOT NULL REFERENCES doctor (id),
    patient_id INT UNSIGNED NOT NULL REFERENCES patient (id),
    case_id INT UNSIGNED NOT NULL REFERENCES patient_case (id),
    health_condition_id INT UNSIGNED NOT NULL REFERENCES health_condition (id),
    last_visit_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_visit_doctor VARCHAR(200) NOT NULL,
    last_event VARCHAR(1000) NOT NULL,
    last_event_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action_url VARCHAR(1000) NOT NULL,
    PRIMARY KEY (patient_id, doctor_id, case_id),
    INDEX (last_event_time),
    INDEX (doctor_id, last_event_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
