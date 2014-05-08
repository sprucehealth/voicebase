-- Removing other option from photo intake
update potential_answer set status = 'INACTIVE' where potential_answer_tag='a_other_acne_location';

-- Fixing typo with empty state text
update localized_text set ltext = 'No prescriptions tried' where app_text_id = (select app_text_id from question_fields where question_field='empty_state_text' and question_id = (select id from question where question_tag='q_acne_prev_treatment_list'));

-- Adding hyphens for otc 
update localized_text set ltext = 'Have you tried over-the-counter acne treatments?' where app_text_id=(select qtext_app_text_id from question where question_tag='q_acne_prev_otc_treatments');
