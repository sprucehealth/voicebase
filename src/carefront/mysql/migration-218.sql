-- Add text title for medication entry
set @language_id=(select id from languages_supported where language='en');
alter table app_text modify column comment varchar(600);
insert into app_text (app_text_tag) values ('text_medication_entry_q');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, 'Which medications are you allergic to?', (select id from app_text where app_text_tag='text_medication_entry_q'));
update question set qtext_app_text_id=(select id from app_text where app_text_tag='text_medication_entry_q') where question_tag='q_allergic_medication_entry';

-- Update text for current medications 
update localized_text set ltext='Are you currently taking any additional medications?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_current_medications');
update localized_text set ltext='Which medications are you currently taking?' where app_text_id=(select qtext_app_text_id from question where question_tag='q_current_medications_entry');
update question set subtext_app_text_id=null where question_tag='q_current_medications_entry';

-- Update text for subquestion to current medications entry
update localized_text set ltext='How long have you been taking this medication?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_length_current_medication');

-- Add new question to state prescription preference
insert into app_text (app_text_tag) values ('txt_prescription_preference_q'),('txt_generic_only'),('txt_no_preference'),('txt_generic_rx_only_alert'),('txt_prescription_preference_short');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "What's your preference for prescription medications?", (select id from app_text where app_text_tag='txt_prescription_preference_q'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Generic only", (select id from app_text where app_text_tag='txt_generic_only'));
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "No preference", (select id from app_text where app_text_tag='txt_no_preference'));
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Generic Rxs only", (select id from app_text where app_text_tag='txt_generic_rx_only_alert'));
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Prescription Preference", (select id from app_text where app_text_tag='txt_prescription_preference_short'));
insert into question (qtype_id, qtext_app_text_id,qtext_short_text_id, question_tag, required, to_alert, alert_app_text_id) values (
	(select id from question_type where qtype='q_type_single_select'), 
	(select id from app_text where app_text_tag='txt_prescription_preference_q'),
	(select id from app_text where app_text_tag='txt_prescription_preference_short'),
	'q_prescription_preference',
	1,
	1,
	(select id from app_text where app_text_tag='txt_generic_rx_only_alert'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status, to_alert) values 
	(
		(select id from question where question_tag='q_prescription_preference'),
		(select id from app_text where app_text_tag='txt_generic_only'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_generic_only',
		0,
		'ACTIVE',
		1), 
	(
		(select id from question where question_tag='q_prescription_preference'),
		(select id from app_text where app_text_tag='txt_no_preference'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_no_preference',
		1, 
		'ACTIVE',
		0
	);


-- Update text for skin condition in past
update localized_text set ltext="Have you been diagnosed for a skin condition in the past?" where app_text_id=(select qtext_app_text_id from question where question_tag='q_prev_skin_condition_diagnosis');

-- Update text for previous skin condition entry
update localized_text set ltext="What skin condition(s) were you diagnosed with?" where app_text_id=(select qtext_app_text_id from question where question_tag='q_list_prev_skin_condition_diagnosis');

-- Update placeholder text for other skin codition entry
update localized_text set ltext='Type to add another condition' 
		where app_text_id=(select app_text_id from question_fields where question_id = (select id from question where question_tag='q_other_skin_condition_entry') and question_field='placeholder_text');

-- Update question and answers for previous medication q_prev_skin_condition_diagnosis
update localized_text set ltext='Select which, if any, of the following conditions you have been treated for.'
	where app_text_id = (select qtext_app_text_id from question where question_tag='q_other_conditions_acne');
update localized_text set ltext = 'High blood pressure' where app_text_id = (select id from app_text where app_text_tag='txt_high_bp_condition');
update potential_answer set status='INACTIVE' where question_id = (select id from question where question_tag='q_other_conditions_acne');
update potential_answer set ordering=25,status='ACTIVE' where potential_answer_tag='a_other_condition_acne_gastiris';
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_other_conditions_acne'),
	(select id from app_text where app_text_tag='txt_high_bp_condition'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_other_condition_acne_high_bp',
	26,
	'ACTIVE'
);
insert into app_text (app_text_tag) values ('text_intestinal_inflammation');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Intestinal inflammation", (select id from app_text where app_text_tag='text_intestinal_inflammation'));	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_other_conditions_acne'),
	(select id from app_text where app_text_tag='text_intestinal_inflammation'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_other_condition_acne_intestinal_inflammation',
	27,
	'ACTIVE'
);
update potential_answer set status='ACTIVE', ordering=28 where potential_answer_tag='a_other_condition_acne_kidney_condition';
update potential_answer set status='ACTIVE', ordering=29 where potential_answer_tag='a_other_condition_acne_liver_disease';
update potential_answer set status='ACTIVE', ordering=30 where potential_answer_tag='a_other_condition_acne_lupus';
insert into app_text (app_text_tag) values ('text_organ_transplant');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Organ transplant", (select id from app_text where app_text_tag='text_organ_transplant'));	
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values (
	(select id from question where question_tag='q_other_conditions_acne'),
	(select id from app_text where app_text_tag='text_organ_transplant'),
	(select id from answer_type where atype='a_type_multiple_choice'),
	'a_other_condition_acne_organ_transplant',
	31,
	'ACTIVE'
);
update potential_answer set status='ACTIVE', ordering=32 where potential_answer_tag='a_other_condition_acne_polycystic_ovary_syndrome';
update potential_answer set status='ACTIVE', ordering=33 where potential_answer_tag='a_other_condition_acne_none';

-- Update pregnancy question
update localized_text set ltext="Are you pregnant, planning a pregnancy or nursing?" where app_text_id = (select qtext_app_text_id from question where question_tag='q_pregnancy_planning');
insert into app_text (app_text_tag) values ('text_pregnancy_disclaimer');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Many acne medications shouldn't be taken while pregnant or nursing.", (select id from app_text where app_text_tag = 'text_pregnancy_disclaimer'));
update question set subtext_app_text_id=(select id from app_text where app_text_tag='text_pregnancy_disclaimer') where question_tag='q_pregnancy_planning';
update question set qtype_id=(select id from question_type where qtype='q_type_single_select') where question_tag='q_pregnancy_planning';
insert into app_text (app_text_tag) values ('text_no_pregnancy');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "No, I'm not and will notify my doctor if I become pregnant during treatment", (select id from app_text where app_text_tag = 'text_no_pregnancy'));
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='text_no_pregnancy') where potential_answer_tag='a_na_pregnancy_planning';




	

