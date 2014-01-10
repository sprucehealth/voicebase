insert into answer_type (atype) values ('a_type_multiple_choice_none');

insert into app_text (app_text_tag, comment) values ('txt_cysts', 'cysts option for symptoms');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Cysts', (select id from app_text where app_text_tag='txt_cysts'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_acne_symptoms'), (select id from app_text where app_text_tag='txt_cysts'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_cysts', 3, 'ACTIVE');


insert into app_text (app_text_tag, comment) values ('txt_none_of_the_above', 'none of the above multiple choice option');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'None of the above', (select id from app_text where app_text_tag='txt_none_of_the_above'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_acne_symptoms'), (select id from app_text where app_text_tag='txt_none_of_the_above'), (select id from answer_type where atype='a_type_multiple_choice_none'), 'a_symptoms_none', 4, 'ACTIVE');
