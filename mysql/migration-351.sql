update potential_answer set to_alert = 1 where potential_answer_tag='a_na_pregnancy_planning';

update localized_text set ltext='Not Pregnant' where app_text_id = (select id from app_text where app_text_tag='txt_not_pregnant');
