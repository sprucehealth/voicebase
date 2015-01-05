-- Not using references to drug_* tables since these are just for lookups
-- (content not used otherwise), it avoids a lot of back and forth joins,
-- the table is really small, and all the other IDs in the form and route
-- tables come directly from DoseSpot.
ALTER TABLE drug_details MODIFY COLUMN ndc VARCHAR(12);
ALTER TABLE drug_details ADD COLUMN generic_drug_name VARCHAR(250);
ALTER TABLE drug_details ADD COLUMN drug_route VARCHAR(250);
ALTER TABLE drug_details ADD COLUMN drug_form VARCHAR(250);
DROP INDEX ndc ON drug_details;
CREATE INDEX drug_details_name_route ON drug_details (generic_drug_name, drug_route);

ALTER TABLE treatment ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE treatment ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);
-- make the drug references non-null. verified that there's no NULLs in any of the environments
ALTER TABLE treatment MODIFY COLUMN drug_name_id INT UNSIGNED NOT NULL;
ALTER TABLE treatment MODIFY COLUMN drug_route_id INT UNSIGNED NOT NULL;
ALTER TABLE treatment MODIFY COLUMN drug_form_id INT UNSIGNED NOT NULL;

ALTER TABLE dr_treatment_template ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE dr_treatment_template ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);

CREATE UNIQUE INDEX drug_name ON drug_name (name);
CREATE UNIQUE INDEX drug_form ON drug_form (name);
CREATE UNIQUE INDEX drug_route ON drug_route (name);
