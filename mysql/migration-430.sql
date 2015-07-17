-- Map parent to patient (teen) and record if the parent has consented for the teen
-- to be treated by Spruce.
CREATE TABLE patient_parent (
    patient_id INT UNSIGNED NOT NULL,
    parent_patient_id INT UNSIGNED NOT NULL,
    consented BOOL NOT NULL DEFAULT false,
    relationship VARCHAR(128) NOT NULL,
    PRIMARY KEY (patient_id, parent_patient_id),
    CONSTRAINT patient_parent_patient FOREIGN KEY (patient_id) REFERENCES patient (id),
    CONSTRAINT patient_parent_parent FOREIGN KEY (parent_patient_id) REFERENCES patient (id)
);

-- Denormalize this instead of having to always query the patient_parent table
-- since it'll be used often.
ALTER TABLE patient ADD COLUMN has_parental_consent BOOL NOT NULL DEFAULT false;
