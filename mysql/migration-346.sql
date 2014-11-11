update localized_text 
	set ltext = 'Add the other medications you are currently taking.' 
	where app_text_id = (select id from app_text where app_text_tag='txt_other_medications_since_tp_entry');

insert into app_text (app_text_tag, comment) values ('txt_placeholder_tp_difficulty', 'placeholder text');
insert into localized_text (language_id, ltext, app_text_id) values 
	((select id from languages_supported where language='en'), 
		'Optional, but let your doctor know if have any questions about how to use your treatment plan more effectively.',
		(select id from app_text where app_text_tag='txt_placeholder_tp_difficulty'));

insert into question_fields (question_field, question_id, app_text_id)
	values ('placeholder_text', (select id from question where question_tag='q_tp_compliance_difficulty'), 
		(select id from app_text where app_text_tag='txt_placeholder_tp_difficulty'));