update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_comments') where question_tag='q_side_effects_from_tp_explain';
	

