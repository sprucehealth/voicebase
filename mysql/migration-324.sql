SET @en=(select id from languages_supported where language='en');
insert into app_text (app_text_tag, comment) values ('txt_add_condition', 'text for other placeholder text to add another condition');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Type another condition', (select id from app_text where app_text_tag='txt_add_condition'));

insert into question_fields (question_field, question_id, app_text_id) values 
	('other_answer_placeholder_text', 
		(select id from question where question_tag='q_list_prev_skin_condition_diagnosis'), 
		(select id from app_text where app_text_tag='txt_add_condition'));
