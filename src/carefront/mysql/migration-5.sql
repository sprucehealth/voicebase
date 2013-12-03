-- Add missing question relating to other condition infromation intake for acne
start transaction;

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_other_condition_acne'), (select id from app_text where app_text_tag='txt_summary_other_condition_acne'), 'q_other_conditions_acne', 1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_gasitris'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_other_condition_acne_gastiris', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_colitis'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_other_condition_acne_colitis', 1);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_kidney_disease'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_other_condition_acne_kidney_condition', 2);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_lupus'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_other_condition_acne_lupus', 3);



commit;