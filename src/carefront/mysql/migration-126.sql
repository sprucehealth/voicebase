update question set subtext_app_text_id = NULL where question_tag='q_acne_type';
update localized_text set ltext = 'Ocular rosacea' where app_text_id = (select id from app_text where app_text_tag='txt_ocular_rosacea');
update localized_text set ltext = 'Papulopustular rosacea' where app_text_id = (select id from app_text where app_text_tag='txt_papulopstular_rosacea');
update localized_text set ltext = 'Erythematotelangiectatic rosacea' where app_text_id = (select id from app_text where app_text_tag='txt_erythematotelangiectatic_rosacea');

