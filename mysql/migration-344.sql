SET @en=(select id from languages_supported where language='en');

insert into question_fields (question_field, question_id, app_text_id) values 
	('add_text', (select id from question where question_tag='	'), (select id from app_text where app_text_tag='txt_add_medication')),
	('save_button_text', (select id from question where question_tag='q_other_medications_since_tp_entry'), (select id from app_text where app_text_tag='txt_save_changes')),
	('remove_button_text', (select id from question where question_tag='q_other_medications_since_tp_entry'), (select id from app_text where app_text_tag='txt_remove_treatment'));


insert into app_text (app_text_tag, comment) values ('txt_med_hx_describe_changes', 'text side effects from medications');
insert into app_text (app_text_tag, comment) values ('txt_med_hx_describe_changes_short', 'text side effects from medications');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Please describe the changes to your medical history:', (select id from app_text where app_text_tag='txt_med_hx_describe_changes')),
	(@en, 'Comments', (select id from app_text where app_text_tag='txt_med_hx_describe_changes_short'));

update question set qtext_app_text_id = (select id from app_text where app_text_tag='txt_med_hx_describe_changes'), question_tag='q_med_hx_changes_relevant_description' where question_tag='q_med_hx_changes_relevant';


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_med_hx_changes_relevance'),
	  (select id from app_text where app_text_tag='txt_med_hx_changes_relevance_short'),
	  'q_med_hx_changes_relevant',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_med_hx_changes_relevant'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_med_hx_changes_relevant_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_med_hx_changes_relevant'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_med_hx_changes_relevant_no',
		1,
		'ACTIVE'
	);