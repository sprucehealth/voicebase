set @language_id = (select id from languages_supported where language='en');

-- Select otc products tried
insert into app_text (app_text_tag) values ('txt_prev_otc_select');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_prev_otc_select'), "Select which over-the-counter acne treatments you have tried.");
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_multiple_choice'),
		(select id from app_text where app_text_tag='txt_prev_otc_select'),
		(select id from app_text where app_text_tag='txt_otc_tried'),
		'q_acne_prev_otc_select',
		1);
insert into app_text (app_text_tag) values ('txt_acne_free');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_acne_free'), "Acne Free");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_acne_free'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_acne_free',
	0,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_cetaphil');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_cetaphil'), "Cetaphil");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_cetaphil'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_cetaphil',
	1,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_clean_and_clear');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_clean_and_clear'), "Clean and Clear");	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_clean_and_clear'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_clean_clear',
	2,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_clearasil');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_clearasil'), "Clearasil")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_clearasil'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_clearasil',
	3,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_noxzema');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_noxzema'), "Noxzema")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_noxzema'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_noxzema',
	4,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_oxy');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_oxy'), "Oxy")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_oxy'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_oxy',
	5,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_proactiv');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_proactiv'), "Proactiv")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_proactiv'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_proactiv',
	6,
	'ACTIVE'
	);
insert into app_text (app_text_tag) values ('txt_zeno');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_zeno'), "Zeno")	;
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_zeno'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_zeno',
	7,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from answer_type where atype='a_type_multiple_choice_other_free_text'),
	'a_other_prev_acne_otc',
	8,
	'ACTIVE'
	);

insert into app_text (app_text_tag) values ('txt_type_another_treatment');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_type_another_treatment'), "Type another treatment")	;
insert into question_fields (question_field, question_id, app_text_id) values (
	"other_answer_placeholder_text",
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_type_another_treatment'));

insert into question_fields (question_field, question_id, app_text_id) values (
	"other_answer_placeholder_text",
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_type_another_treatment'));


-- What drug name product have you tried
set @parent_question_id = (select id from question where question_tag='q_acne_prev_otc_select');
insert into app_text (app_text_tag) values ('txt_formatted_name_product_tried');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_formatted_name_product_tried'), "What <answer_text> have you tried?");
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, qtext_has_tokens, required) values (
		(select id from question_type where qtype='q_type_free_text'),
		(select id from app_text where app_text_tag='txt_formatted_name_product_tried'),
		'q_acne_otc_product_tried',
		@parent_question_id,
		1,
		1);

-- Are you currently using it
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_currently_using_it'),
		'q_using_prev_acne_otc',
		@parent_question_id,
		1);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_using_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from app_text where app_text_tag='txt_current_using'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_using_prev_otc_yes',
	0,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_using_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from app_text where app_text_tag='txt_not_currently_using'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_using_prev_otc_no',
	1	,
	'ACTIVE'
	);

-- How effective is it
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_how_effective'),
		'q_how_effective_prev_acne_otc',
		@parent_question_id,
		1);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_not'),
	(select id from app_text where app_text_tag='txt_not_effective'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_otc_not',
	0	,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_somewhat'),
	(select id from app_text where app_text_tag='txt_answer_summary_somewhat_effective'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_otc_somewhat',
	1	,
	'ACTIVE'
	);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_how_effective_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_very'),
	(select id from app_text where app_text_tag='txt_answer_summary_very_effective'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_how_effective_prev_acne_otc_very_effective',
	2,
	'ACTIVE'
	);

-- Did it irritate your skin?
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_segmented_control'),
		(select id from app_text where app_text_tag='txt_did_it_irritate_skin'),
		'q_irritate_skin_prev_acne_otc',
		@parent_question_id,
		1);
insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_irritate_skin_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_yes'),
	(select id from app_text where app_text_tag='txt_irritated_skin_summary'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_irritate_skin_prev_acne_otc_yes',
	0,
	'ACTIVE'
	);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_irritate_skin_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_no'),
	(select id from app_text where app_text_tag='txt_not_irritated_skin_summary'),
	(select id from answer_type where atype='a_type_segmented_control'),
	'a_irritate_skin_prev_acne_otc_no',
	1,
	'ACTIVE'
	);

-- Anything else you'd like to tell the doctor about it
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values (
		(select id from question_type where qtype='q_type_free_text'),
		(select id from app_text where app_text_tag='txt_anything_else_tell_doctor'),
		'q_anything_else_prev_acne_otc',
		@parent_question_id,
		1);

insert into question_fields (question_field, question_id, app_text_id) values (
	"placeholder_text",
	(select id from question where question_tag='q_anything_else_prev_acne_otc'),
	(select id from app_text where app_text_tag='txt_optional'));


