set @language_id = (select id from languages_supported where language='en');

insert into answer_type (atype) values ('a_type_multiple_choice_other_free_text');

insert into app_text (app_text_tag) values ('txt_prev_prescriptions_select');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_prev_prescriptions_select'), "Select which acne medications you were prescribed.");
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_multiple_choice'),
		(select id from app_text where app_text_tag='txt_prev_prescriptions_select'),
		(select id from app_text where app_text_tag='txt_short_prev_list_treatment'),
		'q_acne_prev_prescriptions_select',
		1);

insert into app_text (app_text_tag) values ('txt_benzaclin');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_benzaclin'), "BenzaClin");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_benzaclin'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_benzaclin',
	0,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_benzoyl_peroxide');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_benzoyl_peroxide'), "Benzoyl peroxide");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_benzoyl_peroxide'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_benzoyl_peroxide',
	1,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_clindamycin');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_clindamycin'), "Clindamycin");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_clindamycin'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_clindamycin',
	2,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_differin');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_differin'), "Differin")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_differin'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_differin',
	3,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_duac');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_duac'), "Duac")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_duac'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_duac',
	4,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_epiduo');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_epiduo'), "Epiduo")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_epiduo'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_epiduo',
	5,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_metrogel');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_metrogel'), "Metrogel")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_metrogel'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_metrogel',
	6,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_minocycline');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_minocycline'), "Minocycline")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_minocycline'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_minocycline',
	7,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_retina_or_tretinoin');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_retina_or_tretinoin'), "Retin-A or Tretinoin")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_retina_or_tretinoin'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_retina_or_tretinoin',
	8,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_tetracycline');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_tetracycline'), "Tetracycline")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_tetracycline'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_tetracycline',
	9,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from answer_type where atype='a_type_multiple_choice_other_free_text'),
	'a_other_prev_acne_prescription',
	10,
	'ACTIVE'
	);

-- Are you currently using it
set @parent_question_id = (select id from question where question_tag='q_acne_prev_prescriptions_select');
insert into app_text (app_text_tag) values ('txt_currently_using_it');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_currently_using_it'), "Are you currently using it?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_currently_using_it'),
		'q_using_prev_acne_prescription',
		@parent_question_id,
		1);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_using_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_using_prev_prescription_yes',
	0,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_using_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_using_prev_prescription_no',
	1	,
	'ACTIVE'
	);

-- How effective is it
insert into app_text (app_text_tag) values ('txt_how_effective');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_how_effective'), "How effective was it?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_how_effective'),
		'q_how_effective_prev_acne_prescription',
		@parent_question_id,
		1);

insert into app_text (app_text_tag) values ('txt_not');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_not'), "Not");
insert into app_text (app_text_tag) values ('txt_not_effective');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_not_effective'), "Not Effective");
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_not'),
	(select id from app_text where app_text_tag='txt_not'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_prescription_not',
	0	,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_somewhat'),
	(select id from app_text where app_text_tag='txt_somewhat'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_prescription_somewhat',
	1	,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_very'),
	(select id from app_text where app_text_tag='txt_very_effective'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_prescription_very_effective',
	2,
	'ACTIVE'
	);


-- Did you use it for more than 3 months
insert into app_text (app_text_tag) values ('txt_did_you_use_for_more_three_months');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_did_you_use_for_more_three_months'), "Did you use it for more than three months?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_did_you_use_for_more_three_months'),
		'q_use_more_three_months_prev_acne_prescription',
		@parent_question_id,
		1);

insert into app_text (app_text_tag) values ('txt_used_more_than_three_months');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_used_more_than_three_months'), "Used for more than 3 months");
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_use_more_three_months_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_use_more_three_months_prev_acne_prescription_yes',
	0,
	'ACTIVE'
	);

insert into app_text (app_text_tag) values ('txt_did_not_use_more_than_three_months');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_did_not_use_more_than_three_months'), "Not used for more than 3 months");
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_use_more_three_months_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_use_more_three_months_prev_acne_prescription_no',
	1,
	'ACTIVE'
	);

-- Did it irritate your skin?
insert into app_text (app_text_tag) values ('txt_did_it_irritate_skin');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_did_it_irritate_skin'), "Did it irritate your skin?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_did_it_irritate_skin'),
		'q_irritate_skin_prev_acne_prescription',
		@parent_question_id,
		1);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_irritate_skin_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_irritate_skin_prev_acne_prescription_yes',
	0,
	'ACTIVE'
	);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_irritate_skin_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_irritate_skin_prev_acne_prescription_no',
	1,
	'ACTIVE'
	);

-- Anything else you'd like to tell the doctor about it
insert into app_text (app_text_tag) values ('txt_anything_else_tell_doctor');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_anything_else_tell_doctor'), "Anything else you'd like to tell the doctor about it?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_free_text'),
		(select id from app_text where app_text_tag='txt_anything_else_tell_doctor'),
		'q_anything_else_prev_acne_prescription',
		@parent_question_id,
		1);
insert into app_text (app_text_tag) values ('txt_optional');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_optional'), "Optional...");

insert into question_fields (question_field, question_id, app_text_id) values (
	"placeholder_text",
	(select id from question where question_tag='q_anything_else_prev_acne_prescription'),
	(select id from app_text where app_text_tag='txt_optional'));





