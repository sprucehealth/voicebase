-- Adding fields for add, save and remove buttons
start transaction;

insert into app_text (app_text_tag, comment) values ('txt_add_button_medication', 'txt for button when adding medication');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_add_button_medication'), 1, "Add Medication");

insert into app_text (app_text_tag, comment) values ('txt_add_button_treatment', 'txt for button when adding treatment');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_add_button_treatment'), 1, "Add Treatment");

insert into app_text (app_text_tag, comment) values ('txt_save_changes', 'txt for saving changes when adding medication or treatment');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_save_changes'), 1, "Save Changes");

insert into app_text (app_text_tag, comment) values ('txt_remove_treatment', 'txt for button to remove treatment');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_remove_treatment'), 1, "Remove Treatment");

insert into app_text (app_text_tag, comment) values ('txt_remove_medication', 'txt for button to remove medication');	
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_remove_medication'), 1, "Remove Medication");

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

commit;