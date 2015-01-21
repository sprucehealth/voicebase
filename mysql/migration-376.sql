
-- The patient_diagnosis, diagnosis_strength and diagnosis_type tables aren't used
DROP TABLE patient_diagnosis;
DROP TABLE diagnosis_strength;
DROP TABLE diagnosis_type;

CREATE TABLE clinical_pathway (
    id INT UNSIGNED AUTO_INCREMENT NOT NULL,
    tag VARCHAR(64) NOT NULL, -- Tag is a unique immutable key for the pathway that can be used in places where the primary key is not appropriate
    name VARCHAR(250) NOT NULL,
    medicine_branch VARCHAR(250) NOT NULL,
    status VARCHAR(32) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (tag)
);

CREATE TABLE clinical_pathway_menu (
    id INT UNSIGNED AUTO_INCREMENT NOT NULL,
    json BLOB NOT NULL,
    status VARCHAR(32) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY (status, created)
);
-- The pathway menu JSON will be a serialized version of PathwayMenu struct defined in common/models_pathways.go

-- health_condition_acne matches the existing tag used for acne
INSERT INTO clinical_pathway (id, tag, name, medicine_branch, status) VALUES (1, 'health_condition_acne', 'Acne', 'Dermatology', 'ACTIVE');

INSERT INTO clinical_pathway_menu (status, created, json)
    VALUES ('ACTIVE', NOW(), '{"title": "What are you here to see the doctor for today?", "items": [{"title": "Acne", "type": "pathway", "pathway": {"id": "1", "tag": "health_condition_acne"}}]}');

-- Denormalize pathway name in case
ALTER TABLE patient_case ADD COLUMN name VARCHAR(250);
UPDATE patient_case SET name = 'Acne';
ALTER TABLE patient_case MODIFY name VARCHAR(250) NOT NULL;

-- Replace health_condition_id with clinical_pathway_id

ALTER TABLE care_providing_state ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE care_providing_state ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE care_providing_state SET clinical_pathway_id = 1;
ALTER TABLE care_providing_state MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE care_providing_state DROP FOREIGN KEY care_providing_state_ibfk_1;
ALTER TABLE care_providing_state DROP COLUMN health_condition_id;

ALTER TABLE app_version_layout_mapping ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE app_version_layout_mapping ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE app_version_layout_mapping SET clinical_pathway_id = 1;
ALTER TABLE app_version_layout_mapping MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE app_version_layout_mapping DROP FOREIGN KEY app_version_layout_mapping_ibfk_1;
ALTER TABLE app_version_layout_mapping DROP COLUMN health_condition_id;

ALTER TABLE diagnosis_layout_version ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE diagnosis_layout_version ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE diagnosis_layout_version SET clinical_pathway_id = 1;
ALTER TABLE diagnosis_layout_version MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE diagnosis_layout_version DROP FOREIGN KEY diagnosis_layout_version_ibfk_3;
ALTER TABLE diagnosis_layout_version DROP COLUMN health_condition_id;

ALTER TABLE doctor_patient_case_feed ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE doctor_patient_case_feed ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE doctor_patient_case_feed SET clinical_pathway_id = 1;
ALTER TABLE doctor_patient_case_feed MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE doctor_patient_case_feed DROP COLUMN health_condition_id;

ALTER TABLE dr_layout_version ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE dr_layout_version ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE dr_layout_version SET clinical_pathway_id = 1;
ALTER TABLE dr_layout_version MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE dr_layout_version DROP FOREIGN KEY dr_layout_version_ibfk_3;
ALTER TABLE dr_layout_version DROP COLUMN health_condition_id;

ALTER TABLE layout_version ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE layout_version ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE layout_version SET clinical_pathway_id = 1;
ALTER TABLE layout_version MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE layout_version DROP FOREIGN KEY layout_version_ibfk_1;
ALTER TABLE layout_version DROP COLUMN health_condition_id;

ALTER TABLE patient_care_provider_assignment ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE patient_care_provider_assignment ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE patient_care_provider_assignment SET clinical_pathway_id = 1;
ALTER TABLE patient_care_provider_assignment MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_care_provider_assignment DROP FOREIGN KEY patient_care_provider_assignment_ibfk_5;
ALTER TABLE patient_care_provider_assignment DROP COLUMN health_condition_id;

ALTER TABLE patient_case ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE patient_case ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE patient_case SET clinical_pathway_id = 1;
ALTER TABLE patient_case MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_case DROP FOREIGN KEY patient_case_ibfk_2;
ALTER TABLE patient_case DROP COLUMN health_condition_id;

ALTER TABLE patient_doctor_layout_mapping ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE patient_doctor_layout_mapping ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE patient_doctor_layout_mapping SET clinical_pathway_id = 1;
ALTER TABLE patient_doctor_layout_mapping MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_doctor_layout_mapping DROP FOREIGN KEY patient_doctor_layout_mapping_ibfk_1;
ALTER TABLE patient_doctor_layout_mapping DROP COLUMN health_condition_id;

ALTER TABLE patient_layout_version ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE patient_layout_version ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE patient_layout_version SET clinical_pathway_id = 1;
ALTER TABLE patient_layout_version MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_layout_version DROP FOREIGN KEY patient_layout_version_ibfk_4;
ALTER TABLE patient_layout_version DROP COLUMN health_condition_id;

ALTER TABLE patient_visit ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE patient_visit ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE patient_visit SET clinical_pathway_id = 1;
ALTER TABLE patient_visit MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE patient_visit DROP FOREIGN KEY patient_visit_ibfk_2;
ALTER TABLE patient_visit DROP COLUMN health_condition_id;

ALTER TABLE section ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE section ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
UPDATE section SET clinical_pathway_id = 1;
ALTER TABLE section MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE section DROP FOREIGN KEY section_ibfk_2;
ALTER TABLE section DROP COLUMN health_condition_id;

-- Denormalize pathway name into doctor_patient_case_feed
ALTER TABLE doctor_patient_case_feed ADD COLUMN clinical_pathway_name VARCHAR(250) NOT NULL;
UPDATE doctor_patient_case_feed SET clinical_pathway_name = 'Acne';

-- Get rid of old health_condition table which should no longer be referenced anywhere
DROP TABLE health_condition;

-- Admin site permissions for pathways
INSERT INTO account_available_permission (name) VALUES ('pathways.view'), ('pathways.edit');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;
INSERT INTO account_group (name) VALUES ('pathways');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'pathways'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'pathways.%';
