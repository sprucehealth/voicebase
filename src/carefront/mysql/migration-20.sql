-- Adding subtext for the one question that needs it

update question set subtext_app_text_id=(select id from app_text where app_text_tag='txt_hint_list_medications') where question_tag='q_current_medications_entry';

