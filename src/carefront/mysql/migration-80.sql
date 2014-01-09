update localized_text set ltext="What's the reason for your visit with Dr. %s today?" where app_text_id = (select qtext_app_text_id from question where id = 1);

insert into app_text (app_text_tag, comment) values ('txt_condition_diagnosis_title', 'text to explain to customer that we are only diagnosing for acne currently');
insert into localized_text (language_id, ltext, app_text_id) values (1, "We're currently only diagnosing and treating acne but will be adding support for more conditions soon.", (select id from app_text where app_text_tag='txt_condition_diagnosis_title'));
update question set qtext_app_text_id=(select id from app_text where app_text_tag='txt_condition_diagnosis_title') where question_tag='q_condition_for_diagnosis';

insert into app_text (app_text_tag, comment) values ('txt_condition_diagnosis_placeholder', 'placeholder text to explain to customer that we are only diagnosing for acne currently');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Help infom what we add next by telling us what your visit today was for...", (select id from app_text where app_text_tag='txt_condition_diagnosis_placeholder'));
insert into question_fields (question_field, question_id, app_text_id) values ('placeholder_text', (select id from question where question_tag='q_condition_for_diagnosis'), (select id from app_text where app_text_tag='txt_condition_diagnosis_placeholder'));

insert into app_text (app_text_tag, comment) values ('txt_submit', 'Submit');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Submit', (select id from app_text where app_text_tag='txt_submit'));
insert into question_fields (question_field, question_id, app_text_id) values ('submit_button_text', (select id from question where question_tag='q_condition_for_diagnosis'), (select id from app_text where app_text_tag='txt_submit'));