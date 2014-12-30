ALTER TABLE answer_type ADD COLUMN deprecated tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE question_type ADD COLUMN deprecated tinyint(1) NOT NULL DEFAULT 0;

UPDATE question_type 
SET deprecated = 1
WHERE qtype IN (
	'q_type_compound', 
	'q_type_multiple_photo', 
	'q_type_photo', 
	'q_type_single_entry', 
	'q_type_single_photo');

UPDATE answer_type
SET deprecated = 1
WHERE atype NOT IN (
	'a_type_multiple_choice', 
	'a_type_segmented_control', 
	'a_type_multiple_choice_none', 
	'a_type_multiple_choice_other_free_text');
