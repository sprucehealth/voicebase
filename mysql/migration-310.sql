update localized_text set ltext='Are you currently taking any medications?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_current_medications');
update question set subtext_app_text_id = NULL where question_tag='q_current_medications';
update potential_answer set atype_id = (select id from answer_type where atype='a_type_multiple_choice_other_free_text') where potential_answer_tag='a_other_skin_iagnosis';
update question set qtype_id = (select id from question_type  where qtype='q_type_multiple_choice') where question_tag='q_skin_description';
