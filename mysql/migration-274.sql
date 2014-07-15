update localized_text set ltext = 'What <parent_answer_text> product have you tried?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_otc_product_tried');
