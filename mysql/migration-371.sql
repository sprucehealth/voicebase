ALTER TABLE dr_favorite_treatment ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE dr_favorite_treatment ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);

ALTER TABLE pharmacy_dispensed_treatment ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE pharmacy_dispensed_treatment ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);

ALTER TABLE unlinked_dntf_treatment ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE unlinked_dntf_treatment ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);

ALTER TABLE requested_treatment ADD COLUMN generic_drug_name_id INT UNSIGNED;
ALTER TABLE requested_treatment ADD FOREIGN KEY (generic_drug_name_id) REFERENCES drug_name (id);
