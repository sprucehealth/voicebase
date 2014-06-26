alter table photo_slot add column placeholder_image_tag varchar(100);

insert into question_type (qtype) values ('q_type_photo_section');


set @language_id = (select id from languages_supported where language='en');
insert into question (qtype_id, qtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_photo_section'),
		(select id from app_text where app_text_tag='txt_face_acne_location'),
		'q_face_photo_section',
		0);

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_photo_section'),
		(select id from app_text where app_text_tag='txt_chest_acne_location'),
		'q_chest_photo_section',
		0);

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_photo_section'),
		(select id from app_text where app_text_tag='txt_chest_acne_location'),
		'q_back_photo_section',
		0);

insert into app_text (app_text_tag) values ('txt_other_location');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_other_location'), "Other Location");

insert into question (qtype_id, qtext_app_text_id, question_tag, required) values 
	((select id from question_type where qtype='q_type_photo_section'),
		(select id from app_text where app_text_tag='txt_other_location'),
		'q_other_location_photo_section',
		0);

insert into photo_slot_type (slot_type) values ('photo_slot_face_front'), ('photo_slot_face_right'), ('photo_slot_face_left'),('photo_slot_other'), ('photo_slot_back'), ('photo_slot_chest');

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_face_photo_section'),
	(select id from app_text where app_text_tag='txt_face_front'),
	(select id from photo_slot_type where slot_type='photo_slot_face_front'),
	'photo_slot_face_front',
	1,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_face_photo_section'),
	(select id from app_text where app_text_tag='txt_profile_left'),
	(select id from photo_slot_type where slot_type='photo_slot_face_left'),
	'photo_slot_face_left',
	1,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_face_photo_section'),
	(select id from app_text where app_text_tag='txt_profile_right'),
	(select id from photo_slot_type where slot_type='photo_slot_face_right'),
	'photo_slot_face_right',
	1,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_face_photo_section'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from photo_slot_type where slot_type='photo_slot_other'),
	'photo_slot_face_other',
	0,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_back_photo_section'),
	(select id from app_text where app_text_tag='txt_back_acne_location'),
	(select id from photo_slot_type where slot_type='photo_slot_back'),
	'photo_slot_back',
	1,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_back_photo_section'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from photo_slot_type where slot_type='photo_slot_other'),
	'photo_slot_other',
	0,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_chest_photo_section'),
	(select id from app_text where app_text_tag='txt_chest_acne_location'),
	(select id from photo_slot_type where slot_type='photo_slot_chest'),
	'photo_slot_chest',
	1,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_chest_photo_section'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from photo_slot_type where slot_type='photo_slot_other'),
	'photo_slot_other',
	0,
	'ACTIVE'
);

insert into photo_slot (
	question_id, 
	slot_name_app_text_id, 
	slot_type_id, 
	placeholder_image_tag,
	required,
	status
) values (
	(select id from question where question_tag='q_other_location_photo_section'),
	(select id from app_text where app_text_tag='txt_other'),
	(select id from photo_slot_type where slot_type='photo_slot_other'),
	'photo_slot_other',
	1,
	'ACTIVE'
);