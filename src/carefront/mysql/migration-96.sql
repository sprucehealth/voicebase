start transaction;
insert into app_text (app_text_tag, comment) values ('txt_select_skin_condition', 'select which skin condition');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Select which skin condition:', (select id from app_text where app_text_tag='txt_select_skin_condition'));
update question set qtext_app_text_id = (select id from app_text where app_text_tag='txt_select_skin_condition') where question_tag='q_list_prev_skin_condition_diagnosis';

commit;