-- Adding fields for each of the autocomplete questions
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into app_text (app_text_tag, comment) values ('txt_add_treatment', 'txt for prompting user to add treatment');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_add_treatment'), 'Add a Treatment');
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_treatment'));

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_treatment'));

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_treatment'));

commit;