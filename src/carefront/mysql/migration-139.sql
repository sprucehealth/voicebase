update localized_text set ltext='Recent changes' where app_text_id =  (select id from app_text where app_text_tag='txt_summary_environment_factors');
update localized_text set ltext='Treatments tried' where app_text_id = (select qtext_short_text_id from question where question_tag='q_acne_prev_treatment_list');
update localized_text set ltext='Skin conditions' where app_text_id = (select qtext_short_text_id from question where question_tag='q_list_prev_skin_condition_diagnosis');
update localized_text set ltext='Gastiritis' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_other_condition_acne_gastiris');
update localized_text set ltext='Skin cancer' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_skin_cancer_diagnosis');
update localized_text set ltext='Face Front' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_face_front_phota_intake');
update localized_text set ltext='None of the above' where app_text_id  = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_planning_pregnancy_none');
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_allergic_medications') where question_tag='q_allergic_medication_entry';
update localized_text set ltext='Are you currently taking any medications (other than those already entered for acne)?' where app_text_id = (select qtext_app_text_id from question where question_tag='qtext_app_text_id');
