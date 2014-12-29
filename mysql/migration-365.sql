-- fixing the potential answer type that was incorrectly specified for 
-- one of the answer types
UPDATE potential_answer
SET atype_id = (SELECT id FROM answer_type WHERE atype='a_type_multiple_choice')
WHERE potential_answer_tag='a_other_condition_acne_liver_disease';