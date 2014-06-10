insert into question_fields (question_field, question_id, app_text_id) values (
	"empty_state_text",
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_treatment_list')
);

insert into question_fields (question_field, question_id, app_text_id) values (
	"empty_state_text",
	(select id from question where question_tag='q_acne_prev_otc_select'),
	(select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_otc_list')
);

