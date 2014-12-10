-- Adding a column to make it easy to identify followup visits within a case
ALTER TABLE patient_visit ADD COLUMN followup tinyint(1) NOT NULL default 0;

UPDATE patient_visit 
SET followup = 1
WHERE sku_id = (SELECT id FROM sku WHERE type='acne_followup');