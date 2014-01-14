insert into app_text (app_text_tag, comment) values ('txt_acne', 'acne');
insert into localized_text (language_id, ltext, app_text_id) values (1, "acne", (select id from app_text where app_text_tag='txt_acne'));

update localized_text set ltext = 'Cystic' where app_text_id = (select id from app_text where app_text_tag='txt_acne_cysts');
update localized_text set ltext = 'Erythematotelangiectatic' where app_text_id = (select id from app_text where app_text_tag='txt_erythematotelangiectatic_rosacea');
update localized_text set ltext = 'Papulopustular' where app_text_id = (select id from app_text where app_text_tag='txt_papulopstular_rosacea');
update localized_text set ltext = 'Ocular' where app_text_id = (select id from app_text where app_text_tag='txt_ocular_rosacea');	

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne') where potential_answer_tag='a_doctor_acne_vulgaris';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_rosacea') where potential_answer_tag='a_doctor_acne_rosacea';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_cysts') where potential_answer_tag='a_acne_cysts';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_inflammatory') where potential_answer_tag='a_acne_inflammatory';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_hormonal') where potential_answer_tag='a_acne_hormonal';

	
	
	
