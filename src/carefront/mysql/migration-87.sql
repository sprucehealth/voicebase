start transaction;

update localized_text set ltext="List the treatments that you've tried for your acne." where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_prev_treatment_list');

insert into app_text (app_text_tag, comment) values ('txt_irritate_skin', 'txt for did this treatment irritate your skin');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Did this treatment irritate your skin?', (select id from app_text where app_text_tag='txt_irritate_skin'));
insert into app_text (app_text_tag, comment) values ('txt_irritated_skin_summary', 'summary text for treatment irritating skin');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Irritated skin', (select id from app_text where app_text_tag='txt_irritated_skin_summary'));
insert into app_text (app_text_tag, comment) values ('txt_not_irritated_skin_summary', 'summary text for treatment irritating skin');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Did not irritate skin', (select id from app_text where app_text_tag='txt_not_irritated_skin_summary') );

insert into question (qtype_id, qtext_app_text_id, question_tag) values ((select id from question_type where qtype='q_type_segmented_control'), (select id from app_text where app_text_tag='txt_irritate_skin'), 'q_treatment_irritate_skin');
update question as q1 inner join question as q2 on q1.id = q2.id set q1.parent_question_id = q2.id where q2.question_tag='q_acne_prev_treatment_list';
insert into potential_answer (question_id, answer_localized_text_id,answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_treatment_irritate_skin'), (select id from app_text where app_text_tag='txt_yes'),(select id from app_text where app_text_tag='txt_irritated_skin_summary'),(select id from answer_type where atype='a_type_segmented_control'), 'a_irritate_skin_yes',0,'ACTIVE');
insert into potential_answer (question_id, answer_localized_text_id,answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_treatment_irritate_skin'), (select id from app_text where app_text_tag='txt_no'),(select id from app_text where app_text_tag='txt_not_irritated_skin_summary'),(select id from answer_type where atype='a_type_segmented_control'), 'a_irritate_skin_no',1,'ACTIVE');
update localized_text set ltext='0-1' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_length_treatment_less_one'); 

commit;