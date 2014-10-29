SET @en=(select id from languages_supported where language='en');

-- Improvements with skin question
insert into app_text (app_text_tag, comment) values ('txt_how_happy', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_how_happy_short', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_very_happy', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_happy', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_neutral', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_unhappy', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_very_unhappy', 'text for how happy patient is with improvements to skin');	

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'How happy are you with the improvements in your skin?', (select id from app_text where app_text_tag='txt_how_happy')),
	(@en, 'Satisfaction level with improvements', (select id from app_text where app_text_tag='txt_how_happy_short')),
	(@en, 'Very happy', (select id from app_text where app_text_tag='txt_very_happy')),
	(@en, 'Happy', (select id from app_text where app_text_tag='txt_happy')),
	(@en, 'Neutral', (select id from app_text where app_text_tag='txt_neutral')),
	(@en, 'Unhappy', (select id from app_text where app_text_tag='txt_unhappy')),
	(@en, 'Very Unhappy', (select id from app_text where app_text_tag='txt_very_unhappy'));

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_how_happy'),
	  (select id from app_text where app_text_tag='txt_how_happy_short'),
	  'q_skin_improvements',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_skin_improvements'),
		(select id from app_text where app_text_tag='txt_very_happy'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_skin_improvements_very_happy',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_improvements'),
		(select id from app_text where app_text_tag='txt_happy'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_skin_improvements_happy',
		1,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_improvements'),
		(select id from app_text where app_text_tag='txt_neutral'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_skin_improvements_neutral',
		2,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_improvements'),
		(select id from app_text where app_text_tag='txt_unhappy'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_skin_improvements_unhappy',
		3,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_improvements'),
		(select id from app_text where app_text_tag='txt_very_unhappy'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_skin_improvements_very_unhappy',
		4,
		'ACTIVE'
	);


-- Why not happy question
insert into app_text (app_text_tag, comment) values ('txt_why_less_than_happy', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_why_less_than_happy_short', 'text for how happy patient is with improvements to skin');
insert into app_text (app_text_tag, comment) values ('txt_patient_chose_not_to_answer', 'Patient chose not to answer');
insert into app_text (app_text_tag, comment) values ('txt_doctor_make_adjustments', 'This will help your doctor make any necessary adjustments to your plan.');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Why aren\'t you happy with the improvements in your skin?' , (select id from app_text where app_text_tag='txt_why_less_than_happy')),
	(@en, 'Comments' , (select id from app_text where app_text_tag='txt_why_less_than_happy_short')),
	(@en, 'Patient chose not to answer' , (select id from app_text where app_text_tag='txt_patient_chose_not_to_answer')),
	(@en, 'This will help your doctor make any necessary adjustments to your plan.', (select id from app_text where app_text_tag='txt_doctor_make_adjustments'));
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_free_text'),
	  (select id from app_text where app_text_tag='txt_why_less_than_happy'),
	  (select id from app_text where app_text_tag='txt_why_less_than_happy_short'),
	  'q_skin_improvements_why_not_happy',
	  1
	);

insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_skin_improvements_why_not_happy'),
		'empty_state_text',
		(select id from app_text where app_text_tag = 'txt_patient_chose_not_to_answer'));


insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_skin_improvements_why_not_happy'),
		'placeholder_text',
		(select id from app_text where app_text_tag = 'txt_doctor_make_adjustments'));


-- Treatment plan compliance
insert into app_text (app_text_tag, comment) values ('txt_using_tp_as_instructed', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_using_tp_as_instructed_short', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_tp_compliance_yes', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_mostly', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_im_not_sure', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_compliant', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_mostly_compliant', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_somewhat_compliant', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_not_compliant', 'text for tp compliance');
insert into app_text (app_text_tag, comment) values ('txt_not_sure', 'text for tp compliance');




insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Overall have you been following your treatment plan as instructed?' , (select id from app_text where app_text_tag='txt_using_tp_as_instructed')),
	(@en, 'Compliance with Treatment Plan' , (select id from app_text where app_text_tag='txt_using_tp_as_instructed_short')),
	(@en, 'Yes, completely' , (select id from app_text where app_text_tag='txt_tp_compliance_yes')),
	(@en, 'Mostly' , (select id from app_text where app_text_tag='txt_mostly')),
	(@en, 'I\'m not sure' , (select id from app_text where app_text_tag='txt_im_not_sure')),
	(@en, 'Compliant' , (select id from app_text where app_text_tag='txt_compliant')),	
	(@en, 'Mostly compliant' , (select id from app_text where app_text_tag='txt_mostly_compliant')),
	(@en, 'Somewhat compliant' , (select id from app_text where app_text_tag='txt_somewhat_compliant')),
	(@en, 'Not compliant' , (select id from app_text where app_text_tag='txt_not_compliant')),
	(@en, 'Not sure' , (select id from app_text where app_text_tag='txt_not_sure'));



insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_using_tp_as_instructed'),
	  (select id from app_text where app_text_tag='txt_using_tp_as_instructed_short'),
	  'q_using_tp_as_instructed',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_using_tp_as_instructed'),
		(select id from app_text where app_text_tag='txt_tp_compliance_yes'),
		(select id from app_text where app_text_tag='txt_compliant'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_tp_as_instructed_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_using_tp_as_instructed'),
		(select id from app_text where app_text_tag='txt_mostly'),
		(select id from app_text where app_text_tag='txt_mostly_compliant'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_tp_as_instructed_mostly',
		1,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_using_tp_as_instructed'),
		(select id from app_text where app_text_tag='txt_sometimes'),
		(select id from app_text where app_text_tag='txt_somewhat_compliant'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_tp_as_instructed_sometimes',
		2,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_using_tp_as_instructed'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from app_text where app_text_tag='txt_not_compliant'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_tp_as_instructed_no',
		3,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_using_tp_as_instructed'),
		(select id from app_text where app_text_tag='txt_im_not_sure'),
		(select id from app_text where app_text_tag='txt_not_sure'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_tp_as_instructed_not_sure',
		4,
		'ACTIVE'
	);


-- Side effects from medications
insert into app_text (app_text_tag, comment) values ('txt_side_effects', 'text side effects from medications');
insert into app_text (app_text_tag, comment) values ('txt_side_effects_short', 'text side effects from medications');


insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Have you experienced any side effects from medications in your treatment plan?' , (select id from app_text where app_text_tag='txt_side_effects')),
	(@en, 'Side effects from medications' , (select id from app_text where app_text_tag='txt_side_effects_short'));

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_side_effects'),
	  (select id from app_text where app_text_tag='txt_side_effects_short'),
	  'q_side_effects_from_tp',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_side_effects_from_tp'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_side_effects_from_tp_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_side_effects_from_tp'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_side_effects_from_tp_no',
		1,
		'ACTIVE'
	);

-- Free text if side effects experienced
insert into app_text (app_text_tag, comment) values ('txt_side_effects_explain', 'text side effects from medications');
insert into app_text (app_text_tag, comment) values ('txt_description', 'text side effects from medications');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Describe the side effects you experienced and which medications caused them.', (select id from app_text where app_text_tag='txt_side_effects_explain')),
	(@en, 'Description.' , (select id from app_text where app_text_tag='txt_description'));
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_free_text'),
	  (select id from app_text where app_text_tag='txt_side_effects_explain'),
	  (select id from app_text where app_text_tag='txt_side_effects_short'),
	  'q_side_effects_from_tp_explain',
	  1
	);
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_side_effects_from_tp_explain'),
		'empty_state_text',
		(select id from app_text where app_text_tag = 'txt_patient_chose_not_to_answer'));
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_side_effects_from_tp_explain'),
		'placeholder_text',
		(select id from app_text where app_text_tag = 'txt_doctor_make_adjustments'));


-- Using all treatments in your plan?
insert into app_text (app_text_tag, comment) values ('txt_using_all_treatments_in_plan', 'using all treatments in plan?');
insert into app_text (app_text_tag, comment) values ('txt_using_all_treatments_in_plan_short', 'using all treatments in plan?');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Are you currently using all of the treatments prescribed in your plan?' , (select id from app_text where app_text_tag='txt_using_all_treatments_in_plan')),
	(@en, 'Using all treatments prescribed in plan' , (select id from app_text where app_text_tag='txt_using_all_treatments_in_plan_short'));


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_using_all_treatments_in_plan'),
	  (select id from app_text where app_text_tag='txt_using_all_treatments_in_plan_short'),
	  'q_using_all_treatments_in_tp',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_using_all_treatments_in_tp'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_all_treatments_in_tp_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_using_all_treatments_in_tp'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_using_all_treatments_in_tp_no',
		1,
		'ACTIVE'
	);

-- treatments patient stopped using
insert into app_text (app_text_tag, comment) values ('txt_treatments_in_tp_stopped_using', 'treamtents in tp that patient stopped using and why');
insert into app_text (app_text_tag, comment) values ('txt_treatments_in_tp_stopped_using_short', 'treamtents in tp that patient stopped using and why');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Which treatments have you stopped using and why?' , (select id from app_text where app_text_tag='txt_treatments_in_tp_stopped_using')),
	(@en, 'Comments' , (select id from app_text where app_text_tag='txt_treatments_in_tp_stopped_using_short'));
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_free_text'),
	  (select id from app_text where app_text_tag='txt_treatments_in_tp_stopped_using'),
	  (select id from app_text where app_text_tag='txt_treatments_in_tp_stopped_using_short'),
	  'q_treatments_in_tp_stopped_using',
	  1
	);
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_treatments_in_tp_stopped_using'),
		'empty_state_text',
		(select id from app_text where app_text_tag = 'txt_patient_chose_not_to_answer'));
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_treatments_in_tp_stopped_using'),
		'placeholder_text',
		(select id from app_text where app_text_tag = 'txt_doctor_make_adjustments'));


-- tp difficulty
insert into app_text (app_text_tag, comment) values ('txt_tp_compliance_difficulty', 'difficulty in complying with treatment plan');
insert into app_text (app_text_tag, comment) values ('txt_tp_compliance_difficulty_short', 'difficulty in complying with treatment plan');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Has any part of your treatment plan been difficult to follow consistently?' , (select id from app_text where app_text_tag='txt_tp_compliance_difficulty')),
	(@en, 'Difficulty complying with treatment plan' , (select id from app_text where app_text_tag='txt_tp_compliance_difficulty_short'));
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_free_text'),
	  (select id from app_text where app_text_tag='txt_tp_compliance_difficulty'),
	  (select id from app_text where app_text_tag='txt_tp_compliance_difficulty_short'),
	  'q_tp_compliance_difficulty',
	  1
	);


-- Started taking any other medications since tp?
insert into app_text (app_text_tag, comment) values ('txt_other_medications_since_tp', 'medications other than prescribed for acne since tp');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Since beginning your treatment plan have you started taking any medications other than the ones prescribed for acne?' , (select id from app_text where app_text_tag='txt_other_medications_since_tp'));

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_other_medications_since_tp'),
	  'q_other_medications_since_tp',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_other_medications_since_tp'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_other_medications_since_tp_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_other_medications_since_tp'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_other_medications_since_tp_no',
		1,
		'ACTIVE'
	);

-- Add other medications currently taking
insert into app_text (app_text_tag, comment) values ('txt_other_medications_since_tp_entry', 'medications other than prescribed for acne since tp');
insert into app_text (app_text_tag, comment) values ('txt_current_medications', 'current medications');
insert into app_text (app_text_tag, comment) values ('txt_no_medications_specified', 'no medications specified');
insert into app_text (app_text_tag, comment) values ('txt_questions_tp_effectively','placeholder text for helping patient use treatment plan more effectively');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Add the other medications are you currently taking' , (select id from app_text where app_text_tag='txt_other_medications_since_tp_entry')),
	(@en, 'Current Medications' , (select id from app_text where app_text_tag='txt_current_medications')),
	(@en, 'No medications specified' , (select id from app_text where app_text_tag='txt_no_medications_specified')),
	(@en, 'Optional, but let your doctor know if have any questions about how to use your treatment plan more effectively.', (select id from app_text where app_text_tag='txt_questions_tp_effectively'));

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_autocomplete'),
	  (select id from app_text where app_text_tag='txt_other_medications_since_tp_entry'),
	  (select id from app_text where app_text_tag='txt_current_medications'),
	  'q_other_medications_since_tp_entry',
	  1
	);
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_other_medications_since_tp_entry'),
		'empty_state_text',
		(select id from app_text where app_text_tag = 'txt_no_medications_specified'));

insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_other_medications_since_tp_entry'),
		'placeholder_text',
		(select id from app_text where app_text_tag = 'txt_questions_tp_effectively'));

insert into question_fields (question_field, question_id, app_text_id) values 
	('add_button_text', (select id from question where question_tag='q_other_medications_since_tp_entry'), (select id from app_text where app_text_tag='txt_add_medication')),
	('add_button_text', (select id from question where question_tag='q_other_medications_since_tp_entry'), (select id from app_text where app_text_tag='txt_add_medication')),
	('placeholder_text', (select id from question where question_tag='q_other_medications_since_tp_entry'), (select id from app_text where app_text_tag='txt_type_add_medication'));


-- Medication allergies since last visit
insert into app_text (app_text_tag, comment) values ('txt_medication_allergies_since_visit', 'medication allergies since last visit');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Since your last visit have you developed any medication allergies? ' , (select id from app_text where app_text_tag='txt_medication_allergies_since_visit'));


insert into question (qtype_id, qtext_app_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_medication_allergies_since_visit'),
	  'q_medication_allergies_since_visit',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_medication_allergies_since_visit'),
		(select id from app_text where app_text_tag='txt_yes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_medication_allergies_since_visit_yes',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_medication_allergies_since_visit'),
		(select id from app_text where app_text_tag='txt_no'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_medication_allergies_since_visit_no',
		1,
		'ACTIVE'
	);

-- Changes to medical history that may be relevant
insert into app_text (app_text_tag, comment) values ('txt_med_hx_changes_relevance', 'changes to medical history that may be relevant');
insert into app_text (app_text_tag, comment) values ('txt_med_hx_changes_relevance_short','changes to medical history that may be relevant');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Are there any changes to your medical history you think may be relevant for your doctor?' , (select id from app_text where app_text_tag='txt_med_hx_changes_relevance')),
	(@en, 'Other changes to medical history' , (select id from app_text where app_text_tag='txt_med_hx_changes_relevance_short'));
insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_free_text'),
	  (select id from app_text where app_text_tag='txt_med_hx_changes_relevance'),
	  (select id from app_text where app_text_tag='txt_med_hx_changes_relevance_short'),
	  'q_med_hx_changes_relevant',
	  1
	);

insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_med_hx_changes_relevant'),
		'empty_state_text',
		(select id from app_text where app_text_tag = 'txt_patient_chose_not_to_answer'));
insert into question_fields (question_id, question_field, app_text_id) values
	((select id from question where question_tag = 'q_med_hx_changes_relevant'),
		'placeholder_text',
		(select id from app_text where app_text_tag = 'txt_doctor_make_adjustments'));


-- Treatment plan section
insert into app_text (app_text_tag, comment) values ('txt_treatment_plan', 'treatment plan');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Treatment Plan' , (select id from app_text where app_text_tag='txt_treatment_plan'));
insert into section (section_title_app_text_id, comment, health_condition_id, section_tag) 
	values ((select id from app_text where app_text_tag = 'txt_treatment_plan'), 'treatment plan section', 1, 'section_treatment_plan');

-- Medication History section in followup
insert into section (section_title_app_text_id, comment, health_condition_id, section_tag) 
	values ((select id from app_text where app_text_tag = 'txt_medical_history'), 'followup medical history section', 1, 'section_followup_medical_history');
























