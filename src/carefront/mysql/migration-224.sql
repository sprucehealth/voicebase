set @language_id=(select id from languages_supported where language='en');
insert into app_text (app_text_tag) values ('txt_otc_tried');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "OTC products tried", (select id from app_text where app_text_tag='txt_otc_tried'));	
update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_otc_tried') where question_tag='q_acne_prev_otc_list';

insert into app_text (app_text_tag) values ('txt_is_pregnant');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Pregnant, planning a pregnancy, or nursing", (select id from app_text where app_text_tag='txt_is_pregnant'));	

insert into app_text (app_text_tag) values ('txt_not_pregnant');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Not pregnant", (select id from app_text where app_text_tag='txt_not_pregnant'));	

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_is_pregnant') where potential_answer_tag='a_yes_pregnancy_planning';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_not_pregnant') where potential_answer_tag='a_na_pregnancy_planning';	



