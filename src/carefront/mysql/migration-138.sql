insert into app_text (app_text_tag, comment) values ('txt_other_location_specified', 'other location specified');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Other location specified", (select id from app_text where app_text_tag='txt_other_location_specified'));
update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_other_location_specified') where question_tag='q_other_acne_location_entry';
