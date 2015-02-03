-- Common diagnoses set for Eczema
INSERT INTO common_diagnosis_set (pathway_id, title) 
VALUES ((SELECT id from clinical_pathway WHERE tag='derm_eczema'), 'Common Eczema Diagnoses');

SET @setID = (SELECT id from common_diagnosis_set);

INSERT INTO common_diagnosis_set_item (diagnosis_code_id, active, common_diagnosis_set_id)
VALUES ('diag_l2084', 1, @setID), ('diag_l209', 1, @setID), ('diag_l400', 1, @setID), ('diag_l409', 1, @setID), 
	   ('diag_l259', 1, @setID), ('diag_l309', 1, @setID), ('diag_l304', 1, @setID), ('diag_b359', 1, @setID), 
	   ('diag_l0101', 1, @setID), ('diag_l739', 1, @setID), ('diag_l089', 1, @setID);

-- Common diagnoses set for Acne
INSERT INTO common_diagnosis_set (pathway_id, title) 
VALUES ((SELECT id from clinical_pathway WHERE tag='health_condition_acne'), 'Common Acne Diagnoses');

SET @setID = (SELECT id from common_diagnosis_set where title='Common Acne Diagnoses');

INSERT INTO common_diagnosis_set_item (diagnosis_code_id, active, common_diagnosis_set_id)
VALUES ('diag_l700', 1, @setID), ('diag_l710', 1, @setID), ('diag_l719', 1, @setID);
	   