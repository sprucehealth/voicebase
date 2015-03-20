-- Each diagnosis_code should be part of each pathway just once
ALTER TABLE common_diagnosis_set_item ADD UNIQUE KEY (diagnosis_code_id, pathway_id);