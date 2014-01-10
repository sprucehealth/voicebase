start transaction;
update localized_text set ltext="Is there anything else you'd like to share about your symptoms with Dr. %s?" where app_text_id = (select qtext_app_text_id from question where question_tag='q_anything_else_acne');
update question set formatted_field_tags="title:doctor_last_name" where question_tag='q_anything_else_acne';
update localized_text set ltext = "This question is optional but feel free to share anything else about your skin that you think the doctor should know..."where app_text_id = (select app_text_id from question_fields where question_id = (select id from question where question_tag='q_anything_else_acne') ); 
commit;