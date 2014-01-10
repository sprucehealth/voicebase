update potential_answer set atype_id = (select id from answer_type where atype='a_type_multiple_choice_none') where potential_answer_tag='a_na_prev_treatment_type';
update localized_text set ltext="What types of treatments have you previously tried for your acne?" where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_prev_treatment_types');
