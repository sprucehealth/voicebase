use carefront_db;

-- Adding fields for each of the autocomplete questions
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into app_text (app_text_tag, comment) values ('txt_add_medication', 'txt for prompting user to add medication');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_add_medication'), 'Add a Medication');
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));

update app_text set app_text_tag='txt_type_add_medication' where app_text_tag='txt_add_medicataion';
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));

insert into app_text (app_text_tag, comment) values ('txt_take_photo_medication', 'txt for prompting user to take a photo of the medication');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_take_photo_medication'), 'Or take a photo of the medication');
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));


insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));


insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));

commit;