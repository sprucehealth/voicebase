update localized_text set ltext='Gastritis' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_other_condition_acne_gastiris');
