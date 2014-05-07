set @language_id=(select id from languages_supported where language='en');

update localized_text set ltext='List the products that you are current using or have tried in the past.' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_prev_otc_list');
update localized_text set ltext = 'Add Medication' where app_text_id = (select id from app_text where app_text_tag='txt_add_treatment');
update localized_text set ltext = 'Type to add a medication' where app_text_id = (select id from app_text where app_text_tag='txt_type_add_treatment');
update localized_text set ltext = 'Add Medication' where app_text_id = (select id from app_text where app_text_tag='txt_add_button_treatment');
update localized_text set ltext = 'Remove Medication' where app_text_id = (select id from app_text where app_text_tag='txt_remove_treatment');


insert into app_text (app_text_tag) values ('txt_add_product'),('txt_remove_product'),('txt_type_to_add_product');
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Add Product", (select id from app_text where app_text_tag='txt_add_product'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Remove Product", (select id from app_text where app_text_tag='txt_remove_product'));	
insert into localized_text (language_id, ltext, app_text_id) values (@language_id, "Type to add a product", (select id from app_text where app_text_tag='txt_type_to_add_product'));	


insert into question_fields (question_field, question_id, app_text_id) values (
		'add_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_add_product')
	),
	(
		'placeholder_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_type_to_add_product')				
	),
	(
		'add_button_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_add_product')				
	),
	(
		'save_button_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_save_changes')				
	),
	(
		'remove_button_text',
		(select id from question where question_tag='q_acne_prev_otc_list'),
		(select id from app_text where app_text_tag='txt_remove_product')				
	);