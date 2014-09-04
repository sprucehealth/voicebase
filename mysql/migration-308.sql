SET @en=(select id from languages_supported where language='en');

-- Skin rating question
insert into app_text (app_text_tag, comment) values ('txt_skin_photo_comparison', 'text for how skin compares based on photos');
insert into app_text (app_text_tag, comment) values ('txt_more_acne_blemishes', 'text for how skin compares based on photos');
insert into app_text (app_text_tag, comment) values ('txt_summary_more_acne_blemishes', 'text for how skin compares based on photos');
insert into app_text (app_text_tag, comment) values ('txt_fewer_acne_blemishes', 'text for how skin compares based on photos');
insert into app_text (app_text_tag, comment) values ('txt_summary_fewer_acne_blemishes', 'text for how skin compares based on photos');
insert into app_text (app_text_tag, comment) values ('txt_about_the_same', 'text for how skin compares based on photos');	
insert into app_text (app_text_tag, comment) values ('txt_summary_about_the_same', 'text for how skin compares based on photos');	
insert into app_text (app_text_tag, comment) values ('txt_short_skin_photo_comparison', 'text for how skin compares based on photos');	

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Compared to the photos you just took, does your skin usually...', (select id from app_text where app_text_tag='txt_skin_photo_comparison')),
	(@en, 'Usual skin compared to photos', (select id from app_text where app_text_tag='txt_short_skin_photo_comparison')),
	(@en, 'Have more acne blemishes', (select id from app_text where app_text_tag='txt_more_acne_blemishes')),
	(@en, 'Has more acne blemishes', (select id from app_text where app_text_tag='txt_summary_more_acne_blemishes')),
	(@en, 'Have fewer acne blemishes', (select id from app_text where app_text_tag='txt_fewer_acne_blemishes')),
	(@en, 'Has fewer acne blemishes', (select id from app_text where app_text_tag='txt_summary_fewer_acne_blemishes')),	
	(@en, 'Look about the same', (select id from app_text where app_text_tag='txt_about_the_same')),
	(@en, 'Looks about the same', (select id from app_text where app_text_tag='txt_summary_about_the_same'));


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_skin_photo_comparison'),
	  (select id from app_text where app_text_tag='txt_short_skin_photo_comparison'),
	  'q_skin_photo_comparison',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_skin_photo_comparison'),
		(select id from app_text where app_text_tag='txt_more_acne_blemishes'),
		(select id from app_text where app_text_tag='txt_summary_more_acne_blemishes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_more_acne_blemishes_photo_comparison',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_photo_comparison'),
		(select id from app_text where app_text_tag='txt_fewer_acne_blemishes'),
		(select id from app_text where app_text_tag='txt_summary_fewer_acne_blemishes'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_fewer_acne_blemishes_photo_comparison',
		1,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_photo_comparison'),
		(select id from app_text where app_text_tag='txt_about_the_same'),
		(select id from app_text where app_text_tag='txt_summary_about_the_same'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_about_the_same_photo_comparison',
		2,
		'ACTIVE'
	);

-- Insurance intake question
insert into app_text (app_text_tag, comment) values ('txt_insurance_coverage', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_insurance_brand_generic', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_insurance_generic_only', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_insurance_idk', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_no_insurance', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_short_insurance_coverage', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_summary_no_insurance', 'text insurance coverage info');
insert into app_text (app_text_tag, comment) values ('txt_summary_insurance_idk', 'text insurance coverage info');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'What type of medications does your insurance cover?', (select id from app_text where app_text_tag='txt_insurance_coverage')),
	(@en, 'Insurance coverage', (select id from app_text where app_text_tag='txt_short_insurance_coverage')),
	(@en, 'Brand name and generic', (select id from app_text where app_text_tag='txt_insurance_brand_generic')),
	(@en, 'Generic only', (select id from app_text where app_text_tag='txt_insurance_generic_only')),
	(@en, 'I don\'t know', (select id from app_text where app_text_tag='txt_insurance_idk')),
	(@en, 'I don\'t have insurance', (select id from app_text where app_text_tag='txt_no_insurance')),
	(@en, 'No insurance', (select id from app_text where app_text_tag='txt_summary_no_insurance')),
	(@en, 'Patient doesn\'t know', (select id from app_text where app_text_tag='txt_summary_insurance_idk'));


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_single_select'),
	  (select id from app_text where app_text_tag='txt_insurance_coverage'),
	  (select id from app_text where app_text_tag='txt_short_insurance_coverage'),
	  'q_insurance_coverage',
	  1
	);

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_insurance_coverage'),
		(select id from app_text where app_text_tag='txt_insurance_brand_generic'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_insurance_brand_generic',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_insurance_coverage'),
		(select id from app_text where app_text_tag='txt_insurance_generic_only'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_insurance_generic_only',
		1,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_insurance_coverage'),
		(select id from app_text where app_text_tag='txt_insurance_idk'),
		(select id from app_text where app_text_tag='txt_summary_insurance_idk'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_insurance_idk',
		2,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_insurance_coverage'),
		(select id from app_text where app_text_tag='txt_no_insurance'),
		(select id from app_text where app_text_tag='txt_summary_no_insurance'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_no_insurance',
		3,
		'ACTIVE'
	);


-- Update title for Photos section
update localized_text set ltext = 'Photos' where app_text_id = 
	(select section_title_app_text_id from section where section_tag='section_photo_diagnosis');


-- Make otc product tried question single free text entry
update question set qtype_id = (select id from question_type where qtype='q_type_single_entry') where question_tag='q_acne_otc_product_tried';

-- Update photo slots names

update photo_slot set slot_name_app_text_id = (select id from app_text where app_text_tag='txt_face_acne_location') 
	where placeholder_image_tag='photo_slot_face_other';
update photo_slot set slot_name_app_text_id = (select id from app_text where app_text_tag='txt_back_acne_location') 
	where placeholder_image_tag='photo_slot_other' and question_id=(select id from question where question_tag='q_back_photo_section');
update photo_slot set slot_name_app_text_id = (select id from app_text where app_text_tag='txt_chest_acne_location') 
	where placeholder_image_tag='photo_slot_other' and question_id=(select id from question where question_tag='q_chest_photo_section');	


-- Update the skin description to include more options

insert into app_text (app_text_tag, comment) values ('txt_sensitive_skin_option', 'text skin description');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Sensitive', (select id from app_text where app_text_tag='txt_sensitive_skin_option'));

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_skin_description'),
		(select id from app_text where app_text_tag='txt_sensitive_skin_option'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_sensitive_skin',
		4,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_skin_description'),
		(select id from app_text where app_text_tag='txt_other'),
		(select id from answer_type where atype='a_type_multiple_choice_other_free_text'),
		'a_other_skin',
		5,
		'ACTIVE'
	);


insert into app_text (app_text_tag, comment) values ('txt_type_another_description', 'placeholder text for adding another skin description');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Type another description', (select id from app_text where app_text_tag='txt_type_another_description'));
insert into question_fields (question_field, question_id, app_text_id) values 
	('other_answer_placeholder_text', (select id from question where question_tag='q_skin_description'), (select id from app_text where app_text_tag='txt_type_another_description'));


-- Update question for when user started getting acne 
update localized_text set ltext = 'When did you start getting acne breakouts?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_onset_acne');

-- Updated title for question acne symtpoms
update localized_text set ltext = 'Has your acne...' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_symptoms');

insert into app_text (app_text_tag, comment) values ('txt_deep_lumps', 'acne symptoms nodules');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Caused deep, hard lumps', (select id from app_text where app_text_tag='txt_deep_lumps'));

update potential_answer set ordering = 16 where potential_answer_tag = 'a_scarring';
update potential_answer set ordering = 17 where potential_answer_tag = 'a_picked_or_squeezed';
update potential_answer set ordering = 18 where potential_answer_tag = 'a_painful_touch';
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_acne_symptoms'),
		(select id from app_text where app_text_tag='txt_deep_lumps'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_deep_lumps',
		19,
		'ACTIVE'
	);
update potential_answer set ordering = 20 where potential_answer_tag = 'a_discoloration';
update potential_answer set ordering = 21 where potential_answer_tag = 'a_created_scars';
update potential_answer set ordering = 22 where potential_answer_tag = 'a_symptoms_none';
update potential_answer set status='INACTIVE' where potential_answer_tag='a_cysts';

-- Update text for acne been getting worse
update localized_text set ltext='Has your acne been getting worse?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_worse');

-- Replace the free text for what could be making your acne worse with a new question that is multiple-select
insert into app_text (app_text_tag, comment) values ('txt_acne_worse_by_something', 'text if acne has been made worse by something');
insert into app_text (app_text_tag, comment) values ('txt_short_acne_worse_by_something', 'text if acne has been made worse by something');
insert into app_text (app_text_tag, comment) values ('txt_diet', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_hair_products', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_makeup', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_hormonal_changes', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_stress', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_sweating_and_sports', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_weather', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_none_or_not_sure', 'options for why acne may be worse');
insert into app_text (app_text_tag, comment) values ('txt_short_none_or_not_sure', 'options for why acne may be worse');

insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Do you think your acne is made worse by any of the following?', (select id from app_text where app_text_tag='txt_acne_worse_by_something')),
	(@en, 'Perceived contributing factors', (select id from app_text where app_text_tag='txt_short_acne_worse_by_something')),
	(@en, 'Diet', (select id from app_text where app_text_tag='txt_diet')),
	(@en, 'Hair Products', (select id from app_text where app_text_tag='txt_hair_products')),
	(@en, 'Makeup', (select id from app_text where app_text_tag='txt_makeup')),
	(@en, 'Hormonal Changes', (select id from app_text where app_text_tag='txt_hormonal_changes')),
	(@en, 'Stress', (select id from app_text where app_text_tag='txt_stress')),
	(@en, 'Sweating or sports', (select id from app_text where app_text_tag='txt_sweating_and_sports')),
	(@en, 'Weather', (select id from app_text where app_text_tag='txt_weather')),
	(@en, 'I\'m not sure', (select id from app_text where app_text_tag='txt_none_or_not_sure')),
	(@en, 'Unsure', (select id from app_text where app_text_tag='txt_short_none_or_not_sure'));

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values
	( (select id from question_type where qtype='q_type_multiple_choice'),
	  (select id from app_text where app_text_tag='txt_acne_worse_by_something'),
	  (select id from app_text where app_text_tag='txt_short_acne_worse_by_something'),
	  'q_acne_worse_contributing_factors',
	  0
	  );

insert into potential_answer (question_id, answer_localized_text_id, answer_summary_text_id, atype_id, potential_answer_tag, ordering, status) values
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_diet'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_diet',
		0,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_hair_products'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_hair_products',
		1,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_makeup'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_makeup',
		2,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_hormonal_changes'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_hormonal_changes',
		3,
		'ACTIVE'
	),
		(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_stress'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_stress',
		4,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_sweating_and_sports'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_sweating_and_sports',
		5,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_weather'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_weater',
		6,	
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_none_or_not_sure'),
		(select id from app_text where app_text_tag='txt_short_none_or_not_sure'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_acne_worse_none_or_not_sure',
		7,
		'ACTIVE'
	),
	(
		(select id from question where question_tag='q_acne_worse_contributing_factors'),
		(select id from app_text where app_text_tag='txt_other'),
		NULL,
		(select id from answer_type where atype='a_type_multiple_choice_other_free_text'),
		'a_acne_worse_other',
		8,
		'ACTIVE'
	);

insert into app_text (app_text_tag, comment) values ('txt_type_another_factor', 'placeholder text for adding another contributing factor');
insert into localized_text (language_id, ltext, app_text_id) values 
	(@en, 'Type another factor', (select id from app_text where app_text_tag='txt_type_another_factor'));
insert into question_fields (question_field, question_id, app_text_id) values 
	('other_answer_placeholder_text', (select id from question where question_tag='q_acne_worse_contributing_factors'), 
	(select id from app_text where app_text_tag='txt_type_another_factor'));


update localized_text set ltext = 'Does getting your period make your acne worse?' 
	where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_worse_period');

update localized_text set ltext = 'Has a doctor ever prescribed medication to treat your acne?' 
	where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_prev_prescriptions');

update localized_text set ltext = 'Not very'
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_how_effective_prev_acne_prescription_not');














