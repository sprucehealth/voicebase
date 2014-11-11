update localized_text
		set ltext = 'Let your doctor know if have any questions about how to use your treatment plan more effectively.'
		where app_text_id = (select id from app_text where app_text_tag='txt_placeholder_tp_difficulty');