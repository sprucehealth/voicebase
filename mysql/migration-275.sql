update potential_answer set potential_answer_tag = 'a_panoxyl' where potential_answer_tag = 'a_anoyl';
update localized_text set ltext='PanOxyl' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_panoxyl');
