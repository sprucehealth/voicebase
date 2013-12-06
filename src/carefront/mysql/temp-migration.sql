use database_3991;
-- Removing the subtext from a question where it should not be present
update question set subtext_app_text_id = NULL where question_tag='q_changes_acne_worse';