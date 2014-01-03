-- Adding fields for each of the autocomplete questions
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'add_text', (select id from app_text where app_text_tag = 'txt_add_treatment'));

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_type_add_treatment'));

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), 'add_photo_text', (select id from app_text where app_text_tag = 'txt_take_photo_treatment'));

insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), (select id from app_text where app_text_tag = 'txt_add_button_treatment'), 'add_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), (select id from app_text where app_text_tag = 'txt_save_changes'), 'save_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_acne_prev_treatment_list'), (select id from app_text where app_text_tag = 'txt_remove_treatment'), 'remove_button_text');

insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_current_medications_entry'), (select id from app_text where app_text_tag = 'txt_add_button_medication'), 'add_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_current_medications_entry'), (select id from app_text where app_text_tag = 'txt_save_changes'), 'save_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_current_medications_entry'), (select id from app_text where app_text_tag = 'txt_remove_treatment'), 'remove_button_text');

insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_allergic_medication_entry'), (select id from app_text where app_text_tag = 'txt_add_button_medication'), 'add_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_allergic_medication_entry'), (select id from app_text where app_text_tag = 'txt_save_changes'), 'save_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_allergic_medication_entry'), (select id from app_text where app_text_tag = 'txt_remove_treatment'), 'remove_button_text');

insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), (select id from app_text where app_text_tag = 'txt_add_button_medication'), 'add_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), (select id from app_text where app_text_tag = 'txt_save_changes'), 'save_button_text');
insert into question_fields (question_id, app_text_id, question_field) values ((select id from question where question_tag = 'q_topical_allergies_medication_entry'), (select id from app_text where app_text_tag = 'txt_remove_treatment'), 'remove_button_text');

insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_changes_acne_worse'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_examples_changes_acne_worse'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_anything_else_acne'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_hint_anything_else_acne_treatment'));

commit;