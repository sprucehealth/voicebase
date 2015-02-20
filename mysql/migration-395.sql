-- CREATE self referencial colmn and FK for parent reference
ALTER TABLE dr_favorite_treatment_plan 
ADD COLUMN parent_id INT(10) UNSIGNED,
ADD CONSTRAINT fk_parent_treatment_plan_id 
FOREIGN KEY(parent_id)
REFERENCES dr_favorite_treatment_plan(id);

-- CREATE the concept of creator ID and the related FK
ALTER TABLE dr_favorite_treatment_plan 
ADD COLUMN creator_id INT(10) UNSIGNED,
ADD CONSTRAINT fk_creator_id_doctor 
FOREIGN KEY(creator_id)
REFERENCES doctor(id);

-- Back fill the creator ID with the associated Dr. ID
UPDATE dr_favorite_treatment_plan SET creator_id = doctor_id;

-- CREATE the memberships table
CREATE TABLE dr_favorite_treatment_plan_membership (
  id int(10) unsigned NOT NULL AUTO_INCREMENT,
  dr_favorite_treatment_plan_id int(10) unsigned NOT NULL,
  doctor_id int(10) unsigned NOT NULL,
  clinical_pathway_id int(10) unsigned NOT NULL,
  PRIMARY KEY (id),
  KEY doctor_id (doctor_id),
  UNIQUE KEY plan_doctor_clinical_pathway (dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id),
  CONSTRAINT dr_favorite_treatment_plan_membership_doctor_id FOREIGN KEY (doctor_id) REFERENCES doctor (id),
  CONSTRAINT dr_favorite_treatment_plan_membership_clinical_pathway_id FOREIGN KEY (clinical_pathway_id) REFERENCES clinical_pathway (id),
  CONSTRAINT dr_favorite_treatment_plan_membership_plan_id FOREIGN KEY (dr_favorite_treatment_plan_id) REFERENCES dr_favorite_treatment_plan (id)
);

-- Backfill memberships for all existing favorite treatment plans
INSERT IGNORE INTO dr_favorite_treatment_plan_membership (dr_favorite_treatment_plan_id, doctor_id, clinical_pathway_id) 
  SELECT id, doctor_id, clinical_pathway_id
  FROM dr_favorite_treatment_plan;

-- DROP FK's, Indexes, and Columns we don't need anymore
ALTER TABLE dr_favorite_treatment_plan DROP FOREIGN KEY dr_favorite_treatment_plan_ibfk_1;
ALTER TABLE dr_favorite_treatment_plan DROP FOREIGN KEY dr_favorite_treatment_plan_ibfk_2;
ALTER TABLE dr_favorite_treatment_plan DROP INDEX doctor_clinical_pathway;
ALTER TABLE dr_favorite_treatment_plan DROP INDEX clinical_pathway_id;
ALTER TABLE dr_favorite_treatment_plan DROP COLUMN doctor_id;
ALTER TABLE dr_favorite_treatment_plan DROP COLUMN clinical_pathway_id;