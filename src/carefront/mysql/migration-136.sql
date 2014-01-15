start transaction;
insert into app_text (app_text_tag, comment) values ('txt_current_medications_yes_no', 'are you currently taking any medications');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Are you currently taking any medications?", (select id from app_text where app_text_tag='txt_current_medications_yes_no'));

insert into app_text (app_text_tag, comment) values ('txt_list_other_than_acne', 'List any other than those you may be using for acne.');
insert into localized_text (language_id, ltext, app_text_id) values (1, "List any other than those you may be using for acne.", (select id from app_text where app_text_tag='txt_list_other_than_acne'));

insert into app_text (app_text_tag, comment) values ('txt_none', 'none');
insert into localized_text (language_id, ltext, app_text_id) values (1, "None", (select id from app_text where app_text_tag='txt_none'));

insert into question (qtype_id, qtext_app_text_id,subtext_app_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_current_medications_yes_no'), (select id from app_text where app_text_tag='txt_list_other_than_acne'), 'q_current_medications', 1);

insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_localized_text_id, ordering, status) values ((select id from question where question_tag='q_current_medications'), (select id from answer_type where atype='a_type_multiple_choice'),'a_current_medications_yes',(select id from app_text where app_text_tag='txt_yes'),0,'ACTIVE');
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_localized_text_id, answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_current_medications'), (select id from answer_type where atype='a_type_multiple_choice'),'a_current_medications_no',(select id from app_text where app_text_tag='txt_no'),(select id from app_text where app_text_tag='txt_none'),1,'ACTIVE');

commit;