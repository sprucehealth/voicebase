set @language_id=(select id from languages_supported where language='en');
insert into app_text (app_text_tag) values ('txt_empty_state_q_acne_prev_otc_list');
update localized_text set ltext = 'No products tried' where app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_otc_list');


insert into question_fields (question_field, question_id, app_text_id) values (
		'empty_state_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_otc_list')
	);