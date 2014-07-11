update question
	set formatted_field_tags = NULL
	where question_tag = 'q_anything_else_acne';

update localized_text
	set ltext = "Is there anything else you'd like to share about your skin?"
		where app_text_id = (select qtext_app_text_id from question where question_tag = 'q_anything_else_acne');


set @language_id = (select id from languages_supported where language='en');

update potential_answer set ordering = 9 where potential_answer_tag = 'a_acne_free';

insert into app_text (app_text_tag) values ('txt_aveeno');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_aveeno'), "Aveeno");
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_aveeno'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_aveeno',
	10,
	'ACTIVE'
	);
update potential_answer set ordering = 11 where potential_answer_tag = 'a_cetaphil';
update potential_answer set ordering = 12 where potential_answer_tag = 'a_clean_clear';
update potential_answer set ordering = 13 where potential_answer_tag = 'a_clearasil';
update potential_answer set ordering = 14 where potential_answer_tag = 'a_noxzema';
update potential_answer set ordering = 15 where potential_answer_tag = 'a_oxy';

insert into app_text (app_text_tag) values ('txt_panoyl');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_panoyl'), "PanOyl");
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_panoyl'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_anoyl',
	16,
	'ACTIVE'
	);

update potential_answer set ordering = 17 where potential_answer_tag = 'a_proactiv';
update potential_answer set ordering = 18 where potential_answer_tag = 'a_other_prev_acne_otc';
update potential_answer set status = 'INACTIVE' where potential_answer_tag in ('a_zeno');


update question
	set subtext_app_text_id = NULL
	where question_tag = 'q_diagnosis_reason_not_suitable';
	
