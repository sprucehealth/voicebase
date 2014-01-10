start transaction;
insert into app_text (app_text_tag, comment) values ('txt_other_acne_location_prompt', 'question title for other acne location');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Acne mainly occurs on the face, neck, chest and back.\n\nIf the doctor determines that you have a condition other than acne you may be asked to visit a local dermatologist's office.", (select id from app_text where app_text_tag='txt_other_acne_location_prompt'));

insert into app_text (app_text_tag, comment) values ('txt_type_add_location', 'type to add a location');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Type to add a location...', (select id from app_text where app_text_tag='txt_type_add_location'));

insert into question (qtype_id, qtext_app_text_id, question_tag) values ((select id from question_type where qtype='q_type_single_entry'), (select id from app_text where app_text_tag='txt_other_acne_location_prompt'), 'q_other_acne_location_entry');
insert into potential_answer (question_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_acne_location_entry'), (select id from answer_type where atype='a_type_single_entry'),'a_other_acne_location_entry',0,'ACTIVE');
insert into question_fields (question_field, question_id, app_text_id) values ('placeholder_text', (select id from question where question_tag='q_other_acne_location_entry'), (select id from app_text where app_text_tag='txt_type_add_location'));

commit;