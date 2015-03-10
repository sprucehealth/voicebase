-- Update the common_diagnosis_set table to only allow a single set to exist
-- per pathway
ALTER TABLE common_diagnosis_set_item ADD COLUMN pathway_id INT UNSIGNED;
UPDATE common_diagnosis_set_item
  SET pathway_id = (SELECT pathway_id FROM common_diagnosis_set as cds WHERE cds.id = common_diagnosis_set_id);

ALTER TABLE common_diagnosis_set_item MODIFY COLUMN pathway_id INT UNSIGNED NOT NULL;
ALTER TABLE common_diagnosis_set_item DROP FOREIGN KEY common_diagnosis_set_item_ibfk_1;
ALTER TABLE common_diagnosis_set_item DROP KEY common_diagnosis_set_id;
ALTER TABLE common_diagnosis_set_item DROP COLUMN common_diagnosis_set_id;
ALTER TABLE common_diagnosis_set_item ADD FOREIGN KEY (pathway_id) REFERENCES clinical_pathway(id);
ALTER TABLE common_diagnosis_set_item ADD KEY (pathway_id, active);

ALTER TABLE common_diagnosis_set DROP COLUMN id;
ALTER TABLE common_diagnosis_set ADD PRIMARY KEY (pathway_id);
