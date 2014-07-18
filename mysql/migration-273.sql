set @language_id = (select id from languages_supported where language='en');
insert into app_text (app_text_tag) values ('txt_doxycycline');



update potential_answer set ordering = 11 where potential_answer_tag = 'a_benzaclin';
update potential_answer set ordering = 12 where potential_answer_tag = 'a_benzoyl_peroxide';
update potential_answer set ordering = 13 where potential_answer_tag = 'a_clindamycin';
update potential_answer set ordering = 14 where potential_answer_tag = 'a_differin';

insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_doxycycline'), 'Doxycycline');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_doxycycline'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_doxycycline',
	15,
	'ACTIVE'
	);
update potential_answer set ordering = 16 where potential_answer_tag = 'a_duac';
update potential_answer set ordering = 17 where potential_answer_tag = 'a_epiduo';
update potential_answer set ordering = 18 where potential_answer_tag = 'a_metrogel';
update potential_answer set ordering = 19 where potential_answer_tag = 'a_minocycline';
update potential_answer set ordering = 20 where potential_answer_tag = 'a_retina_or_tretinoin';
update potential_answer set ordering = 21 where potential_answer_tag = 'a_tetracycline';
update potential_answer set ordering = 22 where potential_answer_tag = 'a_other_prev_acne_prescription';

update localized_text set ltext = 'Benzoyl Peroxide' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag = 'a_benzoyl_peroxide');



