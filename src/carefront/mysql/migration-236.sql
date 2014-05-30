set @language_id = (select id from languages_supported where language='en');
update localized_text set ltext='Other' where language_id = @language_id and app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_doctor_acne_something_else');

insert into app_text (app_text_tag) values ('txt_not_suitable_spruce');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_not_suitable_spruce'), 'Not Suitable For Spruce');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status)
	values ((select id from question where question_tag='q_acne_diagnosis'), 
		(select id from app_text where app_text_tag = 'txt_not_suitable_spruce'),
		(select id from answer_type where atype='a_type_multiple_choice'),
		'a_doctor_acne_not_suitable_spruce',
		7,
		'ACTIVE');


insert into app_text (app_text_tag) values ('txt_describe_patient_condition');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_describe_patient_condition'), "Describe the patient's condition:");
insert into question (qtype_id, qtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_free_text'),
		(select id from app_text where app_text_tag='txt_describe_patient_condition'),
		'q_diagnosis_describe_condition',
		1);
insert into app_text (app_text_tag) values ('txt_type_diagnosis');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_type_diagnosis'), "Type your diagnosis");
insert into question_fields (question_field, question_id, app_text_id) values 
		('placeholder_text', 
			(select id from question where question_tag='q_diagnosis_describe_condition'), 
			(select id from app_text where app_text_tag='txt_type_diagnosis'));


insert into app_text (app_text_tag) values ('txt_why_visit_not_suitable_spruce');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_why_visit_not_suitable_spruce'), "Why isn't this visit suitable for Spruce?");
insert into app_text (app_text_tag) values ('txt_for_internal_purposes');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_for_internal_purposes'), "For internal purposes only, not shared with patient");


insert into question (qtype_id, qtext_app_text_id,subtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_free_text'),
		(select id from app_text where app_text_tag='txt_why_visit_not_suitable_spruce'),
		(select id from app_text where app_text_tag='txt_for_internal_purposes'),
		'q_diagnosis_reason_not_suitable',
		1);
insert into app_text (app_text_tag) values ('txt_describe_why_not_able_to_treat');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_describe_why_not_able_to_treat'), "Describe why you're not able to treat this case");
insert into question_fields (question_field, question_id, app_text_id) values 
		('placeholder_text', 
			(select id from question where question_tag='q_diagnosis_reason_not_suitable'), 
			(select id from app_text where app_text_tag='txt_describe_why_not_able_to_treat'));






