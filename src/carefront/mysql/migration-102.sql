start transaction;

update question set required=1 where question_tag='q_onset_acne';
update question set required=1 where question_tag='q_acne_worse';	
update question set required=1 where question_tag='q_acne_symptoms';	
update question set required=1 where question_tag='q_acne_worse_period';	
update question set required=1 where question_tag='q_periods_regular';	
update question set required=1 where question_tag='q_skin_description';	
update question set required=1 where question_tag='q_acne_prev_treatment_types';	
update question set required=1 where question_tag='q_effective_treatment';	
update question set required=1 where question_tag='q_using_treatment';	
update question set required=1 where question_tag='q_length_treatment';	
update question set required=1 where question_tag='q_treatment_irritate_skin';	
update question set required=1 where question_tag='q_acne_location';
update question set required=1 where question_tag='q_other_acne_location_entry';
update question set required=1 where question_tag='q_chest_photo_intake';
update question set required=1 where question_tag='q_face_photo_intake';
update question set required=1 where question_tag='q_other_photo_intake';
update question set required=1 where question_tag='q_back_photo_intake';
update question set required=1 where question_tag='q_pregnancy_planning';
update question set required=1 where question_tag='q_allergic_medications';
update question set required=1 where question_tag='q_allergic_medication_entry';
update question set required=1 where question_tag='q_length_current_medication';
update question set required=1 where question_tag='q_prev_skin_condition_diagnosis';
update question set required=1 where question_tag='q_list_prev_skin_condition_diagnosis';
update question set required=1 where question_tag='q_other_conditions_acne';

update question set required=0 where question_tag='q_changes_acne_worse';	
update question set required=0 where question_tag='q_acne_prev_treatment_list';	
update question set required=0 where question_tag='q_anything_else_acne';	
update question set required=0 where question_tag='q_current_medications_entry';	

commit;