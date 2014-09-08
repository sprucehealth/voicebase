SET @en=(select id from languages_supported where language='en');
insert into app_text (app_text_tag, comment) values ('txt_neutrogena', 'option for skin description');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Neutrogena', (select id from app_text where app_text_tag='txt_neutrogena'));



update potential_answer set ordering=19  where potential_answer_tag = 'a_acne_free';
update potential_answer set ordering=20  where potential_answer_tag = 'a_aveeno';
update potential_answer set ordering=21  where potential_answer_tag = 'a_cetaphil';
update potential_answer set ordering=22  where potential_answer_tag = 'a_clean_clear';
update potential_answer set ordering=23  where potential_answer_tag = 'a_clearasil';
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_acne_prev_otc_select'),
		(select id from app_text where app_text_tag='txt_neutrogena'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_neutrogena',
		24,
		'ACTIVE'
	);
update potential_answer set ordering=25  where potential_answer_tag = 'a_noxzema';
update potential_answer set ordering=26  where potential_answer_tag = 'a_oxy';
update potential_answer set ordering=27  where potential_answer_tag = 'a_panoxyl';
update potential_answer set ordering=28  where potential_answer_tag = 'a_proactiv';
update potential_answer set ordering=29  where potential_answer_tag = 'a_other_prev_acne_otc';

	



