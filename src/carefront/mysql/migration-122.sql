update localized_text set ltext='Gastiritis' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='answer_localized_text_id');
update localized_text set ltext='Comedonal' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag ='a_acne_comedonal');
