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

INSERT INTO doctor_patient_case_feed (
    doctor_id, patient_id, case_id, health_condition_id, last_visit_time, last_visit_doctor,
    last_event, last_event_time, action_url)
SELECT a.role_type_id, c.patient_id, a.patient_case_id, 1, MAX(COALESCE(v.submitted_date, v.creation_date)),
    d.short_display_name, '', MAX(COALESCE(v.submitted_date, v.creation_date)),
    'spruce:///action/view_case?case_id=' || a.patient_case_id
FROM patient_case_care_provider_assignment a
INNER JOIN patient_case c ON c.id = a.patient_case_id
INNER JOIN doctor d ON d.id = a.provider_id
INNER JOIN role_type rt ON rt.id = a.role_type_id
INNER JOIN patient_visit v ON v.patient_case_id = a.patient_case_id
WHERE rt.role_type_tag = 'DOCTOR'
    AND NOT v.status IN ('PENDING', 'OPEN')
GROUP BY a.role_type_id, c.patient_id, a.patient_case_id, d.short_display_name;
