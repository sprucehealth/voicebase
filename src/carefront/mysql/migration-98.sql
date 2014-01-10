start transaction;
update question set qtype_id = (select id from question_type where qtype='q_type_multiple_choice') where question_tag='q_other_conditions_acne';
update localized_text set ltext = 'Kidney disease' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_other_condition_acne_kidney_condition');
update localized_text set ltext = 'Liver disease' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_liver_disease_diagnosis');
update localized_text set ltext = 'Polycystic ovary syndrome' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_other_condition_acne_polycystic_ovary_syndrome');
commit;