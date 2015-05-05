-- Update indexes based on auditing queries against the table
CREATE UNIQUE INDEX case_role_provider ON patient_case_care_provider_assignment (patient_case_id, role_type_id, provider_id);
CREATE INDEX role_provider_status ON patient_case_care_provider_assignment (role_type_id, provider_id, status);
DROP INDEX role_type_id ON patient_case_care_provider_assignment;
DROP INDEX patient_case_id ON patient_case_care_provider_assignment;

-- Remove unecessary index
DROP INDEX provider_role_id ON patient_care_provider_assignment;

-- Bizarrly the foreign key constraint didn't exist
CREATE INDEX case_status ON patient_visit (patient_case_id, status, submitted_date);
ALTER TABLE patient_visit
    ADD CONSTRAINT fk_patient_visit_patient_case_id
        FOREIGN KEY (patient_case_id)
        REFERENCES patient_case (id);

DROP TABLE doctor_patient_case_feed;
