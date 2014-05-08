update potential_answer set status='INACTIVE' where question_id=(select id from question where question_tag='q_pregnancy_planning');
update potential_answer set status='ACTIVE' where potential_answer_tag in ('a_yes_pregnancy_planning', 'a_na_pregnancy_planning');
update localized_text set ltext='Gastritis' where app_text_id=(select id from app_text where app_text_tag='txt_gasitris');