insert into app_text (app_text_tag, comment) values ('txt_sometimes', 'sometimes');
insert into localized_text (language_id, ltext, app_text_id) values 
	((select id from languages_supported where language='en'), 
		'Sometimes',
		(select id from app_text where app_text_tag='txt_sometimes'));
update potential_answer
		set answer_localized_text_id = (select id from app_text where app_text_tag='txt_sometimes')
		where potential_answer_tag='a_using_tp_as_instructed_sometimes';

