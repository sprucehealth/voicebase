set @language_id=(select id from languages_supported where language='en');

update potential_answer set status='ACTIVE' where potential_answer_tag='a_combination_skin';
update localized_text set ltext='When did you start getting breakouts?' where app_text_id=(select qtext_app_text_id from question where question_tag='q_onset_acne');

-- New question for "Have your breakouts..."
update localized_text set ltext='Have your breakouts...' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_symptoms');
insert into app_text (app_text_tag) values ('txt_picked_or_squeezed');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Been picked or squeezed", (select id from app_text where app_text_tag='txt_picked_or_squeezed'));	
insert into app_text (app_text_tag) values ('txt_created_scars');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Created scars", (select id from app_text where app_text_tag='txt_created_scars'));	

update localized_text set ltext='Been painful to the touch' where app_text_id=(select answer_localized_text_id from potential_answer where potential_answer_tag='a_painful_touch');
update localized_text set ltext='Turned into cysts' where app_text_id=(select answer_localized_text_id from potential_answer where potential_answer_tag='a_cysts');
update localized_text set ltext='Caused discoloration' where app_text_id=(select answer_localized_text_id from potential_answer where potential_answer_tag='a_discoloration');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status, to_alert) values 
	(
		(select id from question where question_tag='q_acne_symptoms'),
		(select id from app_text where app_text_tag='txt_picked_or_squeezed'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_picked_or_squeezed',
		10,
		'ACTIVE',
		0), 
	(
		(select id from question where question_tag='q_acne_symptoms'),
		(select id from app_text where app_text_tag='txt_created_scars'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_created_scars',
		14, 
		'ACTIVE',
		0
	);
update potential_answer set ordering=11 where potential_answer_tag='a_painful_touch';
update potential_answer set ordering=12 where potential_answer_tag='a_cysts';
update potential_answer set ordering=13 where potential_answer_tag='a_discoloration';


update localized_text set ltext = "Have your breakouts been getting worse?" where app_text_id=(select qtext_app_text_id from question where question_tag='q_acne_worse');
update localized_text set ltext = "Are there any recent changes that could be affecting your skin?" where app_text_id=(select qtext_app_text_id from question where question_tag='q_changes_acne_worse');
update localized_text set ltext = "Ex: new cosmetics, sports, warmer weather, increased stress." 
	where app_text_id=(select app_text_id from question_fields where question_id=(select id from question where question_tag='q_changes_acne_worse') and question_field='placeholder_text');

update localized_text set ltext = "Does getting your period make your breakouts worse?" where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_worse_period');

insert into app_text (app_text_tag) values ('txt_acne_prev_prescriptions_q'),('txt_tried_otc_acne'),('txt_list_otc_products');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Have you been prescribed medications to treat your acne?", (select id from app_text where app_text_tag='txt_acne_prev_prescriptions_q'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Have you tried over the counter acne treatments?", (select id from app_text where app_text_tag='txt_tried_otc_acne'));
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "List the products that you are current using or have tried in the past.", (select id from app_text where app_text_tag='txt_list_otc_products'));

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values (
	(select id from question_type where qtype='q_type_single_select'), 
	(select id from app_text where app_text_tag='txt_acne_prev_prescriptions_q'),
	'q_acne_prev_prescriptions',
	1);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status, to_alert) values 
	(
		(select id from question where question_tag='q_acne_prev_prescriptions'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_prev_prescriptions_yes',
		0,
		'ACTIVE',
		0), 
	(
		(select id from question where question_tag='q_acne_prev_prescriptions'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_prev_prescriptions_no',
		1, 
		'ACTIVE',
		0
	);
update localized_text set ltext = "List medications that you are currently using or have tried in the past." where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_prev_treatment_list');
update localized_text set ltext = "Are you currently using this medication?"  where app_text_id = (select qtext_app_text_id from question where question_tag='q_using_treatment');
update localized_text set ltext = "How effective was this medication?"  where app_text_id = (select qtext_app_text_id from question where question_tag='q_effective_treatment');
update localized_text set ltext = "Did this medication irritate your skin?"  where app_text_id = (select qtext_app_text_id from question where question_tag='q_treatment_irritate_skin');
update localized_text set ltext = "Approximately how many months did you use this medication for?"  where app_text_id = (select qtext_app_text_id from question where question_tag='q_length_treatment');
update localized_text set ltext = "Prescription tried" where app_text_id = (select qtext_short_text_id from question where question_tag='q_acne_prev_treatment_list');


update localized_text set ltext = "Is there anything else you'd like to share about your skin with Dr. %s?" where app_text_id = (select qtext_app_text_id from question where question_tag='q_anything_else_acne');
insert into app_text (app_text_tag) values ('txt_placeholder_anything_else_acne');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "This question is optional but is your chance to let the doctor know what's on your mind.", (select id from app_text where app_text_tag='txt_placeholder_anything_else_acne'));	

insert into question_fields (question_field, question_id, app_text_id) values (
	'placeholder_text',
	(select id from question where question_tag='q_anything_else_acne'),
	(select id from app_text where app_text_tag='txt_placeholder_anything_else_acne')
);

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values (
	(select id from question_type where qtype='q_type_single_select'), 
	(select id from app_text where app_text_tag='txt_tried_otc_acne'),
	'q_acne_prev_otc_treatments',
	1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status, to_alert) values 
	(
		(select id from question where question_tag='q_acne_prev_otc_treatments'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_prev_otc_treatments_yes',
		0,
		'ACTIVE',
		0), 
	(
		(select id from question where question_tag='q_acne_prev_otc_treatments'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_prev_otc_treatments_no',
		1, 
		'ACTIVE',
		0
	);

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values (
	(select id from question_type where qtype='q_type_autocomplete'), 
	(select id from app_text where app_text_tag='txt_list_otc_products'),
	'q_acne_prev_otc_list',
	1);

insert into app_text (app_text_tag) values ('txt_using_otc_q'),('txt_effective_otc_q'),('txt_otc_irritate_skin_q'),('txt_length_otc_q');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Are you currently using this product?", (select id from app_text where app_text_tag='txt_using_otc_q'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "How effective was this product?", (select id from app_text where app_text_tag='txt_effective_otc_q'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Did this product irritate your skin?", (select id from app_text where app_text_tag='txt_otc_irritate_skin_q'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Approximately how many months did you use this product for?", (select id from app_text where app_text_tag='txt_length_otc_q'));	

set @parent_question_id=(select id from question where question_tag='q_acne_prev_otc_list');

insert into question (qtype_id, qtext_app_text_id, question_tag,parent_question_id, required) values (
	(select id from question_type where qtype='q_type_segmented_control'), 
	(select id from app_text where app_text_tag='txt_using_otc_q'),
	'q_using_otc',
	@parent_question_id,
	1);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values 
	(
		(select id from question where question_tag='q_using_otc'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from app_text where app_text_tag='txt_answer_summary_using'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_using_otc_yes',
		0,
		'ACTIVE'), 
	(
		(select id from question where question_tag='q_using_otc'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from app_text where app_text_tag='txt_answer_summary_not_using'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_using_otc_no',
		1, 
		'ACTIVE'
	);


insert into question (qtype_id, qtext_app_text_id, question_tag,parent_question_id, required) values (
	(select id from question_type where qtype='q_type_segmented_control'), 
	(select id from app_text where app_text_tag='txt_effective_otc_q'),
	'q_effective_otc',
	@parent_question_id,
	1);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values 
	(
		(select id from question where question_tag='q_effective_otc'),
		(select id from app_text where app_text_tag='txt_not_very'),
		(select id from app_text where app_text_tag='txt_answer_summary_not_effective'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_effective_otc_not_very',
		0,
		'ACTIVE'), 
	(
		(select id from question where question_tag='q_effective_otc'),
		(select id from app_text where app_text_tag='txt_somewhat'),
		(select id from app_text where app_text_tag='txt_answer_summary_somewhat_effective'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_effective_otc_somewhat',
		1, 
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_effective_otc'),
		(select id from app_text where app_text_tag='txt_very'),
		(select id from app_text where app_text_tag='txt_answer_summary_very_effective'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_effective_otc_very',
		2, 
		'ACTIVE'
	);


insert into question (qtype_id, qtext_app_text_id, question_tag,parent_question_id, required) values (
	(select id from question_type where qtype='q_type_segmented_control'), 
	(select id from app_text where app_text_tag='txt_otc_irritate_skin_q'),
	'q_otc_irritate_skin',
	@parent_question_id,
	1);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values 
	(
		(select id from question where question_tag='q_otc_irritate_skin'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from app_text where app_text_tag='txt_irritated_skin_summary'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_otc_irritate_skin_yes',
		0,
		'ACTIVE'), 
	(
		(select id from question where question_tag='q_otc_irritate_skin'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from app_text where app_text_tag='txt_not_irritated_skin_summary'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_otc_irritate_skin_no',
		1, 
		'ACTIVE'
	);

insert into question (qtype_id, qtext_app_text_id, question_tag,parent_question_id, required) values (
	(select id from question_type where qtype='q_type_segmented_control'), 
	(select id from app_text where app_text_tag='txt_length_otc_q'),
	'q_length_otc',
	@parent_question_id,
	1);


insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values 
	(
		(select id from question where question_tag='q_length_otc'),
		(select id from app_text where app_text_tag='txt_one_or_less'),
		(select id from app_text where app_text_tag='txt_answer_summary_less_month'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_length_otc_less_one',
		0,
		'ACTIVE'), 
	(
		(select id from question where question_tag='q_length_otc'),
		(select id from app_text where app_text_tag='txt_two_five_months'),
		(select id from app_text where app_text_tag='txt_answer_summary_two_five_months'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_length_otc_two_five_months',
		1, 
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_length_otc'),
		(select id from app_text where app_text_tag='txt_six_eleven_months'),
		(select id from app_text where app_text_tag='txt_answer_summary_six_eleven_months'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_length_otc_two_six_eleven_months',
		2, 
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_length_otc'),
		(select id from app_text where app_text_tag='txt_twelve_plus_months'),
		(select id from app_text where app_text_tag='txt_answer_summary_twelve_plus_months'),
		(select id from answer_type where atype='a_type_segmented_control'),
		'a_length_otc_twelve_plus_months',
		3, 
		'ACTIVE'
	);




