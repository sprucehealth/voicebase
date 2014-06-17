update localized_text set ltext='Chest' where app_text_id = (select qtext_app_text_id from question where question_tag='q_chest_photo_section');
update question set qtext_app_text_id = (select id from app_text where app_text_tag='txt_back_acne_location') where question_tag='q_back_photo_section';
