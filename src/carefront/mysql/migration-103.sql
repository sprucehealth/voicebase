start transaction;

insert into app_text (app_text_tag, comment) values ('txt_prompt_add_skin_condition', 'type to add a condition');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Type to add a condition...", (select id from app_text where app_text_tag='txt_prompt_add_skin_condition'));

insert into question (qtype_id, question_tag) values ((select id from question_type where qtype='q_type_single_entry'), 'q_other_skin_condition_entry');
insert into potential_answer (question_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_skin_condition_entry'), (select id from answer_type where atype='a_type_single_entry'),'a_other_skin_condition_entry',0,'ACTIVE');
insert into question_fields (question_field, question_id, app_text_id) values ('placeholder_text', (select id from question where question_tag='q_other_skin_condition_entry'), (select id from app_text where app_text_tag='txt_prompt_add_skin_condition'));



commit;