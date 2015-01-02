-- Attempting to remove diagnosis code information from mysql
-- given that we are now storing it in postgres as the source of truth

-- Getting rid of any foreign key constraints to the diagnosis_code table
ALTER TABLE visit_diagnosis_item DROP FOREIGN KEY visit_diagnosis_item_ibfk_2;
ALTER TABLE diagnosis_details_layout DROP FOREIGN KEY diagnosis_details_layout_ibfk_1;
ALTER TABLE diagnosis_details_layout_template DROP FOREIGN KEY diagnosis_details_layout_template_ibfk_1;

-- Getting rid of diagnosis information from mysql
DELETE FROM diagnosis_code;
DROP TABLE diagnosis_use_additional_code_note;
DROP TABLE diagnosis_code_first_note;
DROP TABLE diagnosis_excludes2_note;
DROP TABLE diagnosis_excludes1_note;
DROP TABLE diagnosis_inclusion_term;
DROP TABLE diagnosis_includes_note;
DROP TABLE diagnosis_code;

-- Modify diganosis code ids to be a varchar 
ALTER TABLE visit_diagnosis_item MODIFY COLUMN diagnosis_code_id varchar(32) NOT NULL;
ALTER TABLE diagnosis_details_layout MODIFY COLUMN diagnosis_code_id varchar(32) NOT NULL;
ALTER TABLE diagnosis_details_layout_template MODIFY COLUMN diagnosis_code_id varchar(32) NOT NULL;
