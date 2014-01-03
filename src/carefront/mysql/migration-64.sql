-- Adding fields for each of the autocomplete questions
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_allergic_medication_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));


insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_current_medications_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));


insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_medication'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_medication'));


commit;