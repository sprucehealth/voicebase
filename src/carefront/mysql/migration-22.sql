-- Making potential_answer_id nullable because it is not necessary for a patient info intake to have a reference to a potential_answer_id 
-- like in the situation of a free_text or autocomplete answer
alter table patient_info_intake modify potential_answer_id int unsigned;