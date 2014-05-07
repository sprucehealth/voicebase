set @language_id=(select id from languages_supported where language='en');

insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "No products tried", (select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_otc_list'));	
