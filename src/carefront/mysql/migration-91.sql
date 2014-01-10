start transaction;
update potential_answer set status='INACTIVE' where potential_answer_tag='a_yes_pregnancy_planning';
update potential_answer set status='INACTIVE' where potential_answer_tag='a_na_pregnancy_planning';
update question set qtype_id = (select id from question_type where qtype='q_type_multiple_choice') where question_tag='q_pregnancy_planning';

insert into app_text (app_text_tag, comment) values ('txt_pregnant', 'option to indicate that the patient is pregnant');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Pregnant', (select id from app_text where app_text_tag='txt_pregnant'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_pregnancy_planning'), (select id from app_text where app_text_tag='txt_pregnant'), (select id from answer_type where atype='a_type_multiple_choice'),'a_pregnant',2,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_nursing', 'option to indicate that the patient is nursing');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Nursing', (select id from app_text where app_text_tag='txt_nursing'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_pregnancy_planning'), (select id from app_text where app_text_tag='txt_nursing'), (select id from answer_type where atype='a_type_multiple_choice'),'a_nursing',3,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_planning_pregnancy', 'option to indicate that the patient is planning a pregnancy');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Planning a pregnancy', (select id from app_text where app_text_tag='txt_planning_pregnancy'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_pregnancy_planning'), (select id from app_text where app_text_tag='txt_planning_pregnancy'), (select id from answer_type where atype='a_type_multiple_choice'),'a_planning_pregnancy',4,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_pregnancy_nursing_none', 'option to indicate that the patient neither pregnant nor planning a pregnancy');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'I am not pregnant, planning a pregnancy or nursing.', (select id from app_text where app_text_tag='txt_pregnancy_nursing_none'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_pregnancy_planning'), (select id from app_text where app_text_tag='txt_pregnancy_nursing_none'), (select id from answer_type where atype='a_type_multiple_choice_none'),'a_planning_pregnancy_none',5,'ACTIVE');


update localized_text set ltext='Medical History' where app_text_id = (select section_title_app_text_id from section where section_tag='section_medical_history');

commit;
