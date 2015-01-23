
-- Link ftp to pathway
ALTER TABLE dr_favorite_treatment_plan ADD COLUMN clinical_pathway_id INT UNSIGNED;
ALTER TABLE dr_favorite_treatment_plan ADD FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id);
-- All FTP are for the acne pathway
UPDATE dr_favorite_treatment_plan SET clinical_pathway_id = 1;
ALTER TABLE dr_favorite_treatment_plan MODIFY clinical_pathway_id INT UNSIGNED NOT NULL;

-- Create a composite index for (doctor_id, clinical_pathway_id) so don't need the one just on doctor_id
CREATE INDEX doctor_clinical_pathway ON dr_favorite_treatment_plan (doctor_id, clinical_pathway_id);
DROP INDEX doctor_id ON dr_favorite_treatment_plan;
