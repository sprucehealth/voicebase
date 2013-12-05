-- Adding additional fields for free text
start transaction;
set @en_language_id = (select id from languages_supported where language="en");
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_changes_acne_worse'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_examples_changes_acne_worse'));
insert into question_fields (question_id, question_field, app_text_id) values ((select id from question where question_tag = 'q_anything_else_acne'), 'placeholder_text', (select id from app_text where app_text_tag = 'txt_hint_anything_else_acne_treatment'));
commit;