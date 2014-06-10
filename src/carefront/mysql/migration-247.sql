update localized_text set ltext="Type another treatment name" where app_text_id = (select id from app_text where app_text_tag='txt_type_another_treatment');
insert into question_fields (question_field, question_id, app_text_id) values (
	"other_answer_placeholder_text",
	(select id from question where question_tag='q_acne_prev_prescriptions_select'),
	(select id from app_text where app_text_tag='txt_type_another_treatment')
);

update question set qtype_id = (select id from question_type where qtype='q_type_single_entry') where question_tag in  ('q_anything_else_prev_acne_prescription', 'q_anything_else_prev_acne_otc', 'q_anything_else_prev_acne_otc', 'q_acne_otc_product_tried');
update localized_text set ltext="What <parent_answer_text> have you tried?" where app_text_id = (select id from app_text where app_text_tag='txt_formatted_name_product_tried');

insert into question_fields (question_field, question_id, app_text_id) values (
	"placeholder_text",
	(select id from question where question_tag='q_acne_otc_product_tried'),
	(select id from app_text where app_text_tag='txt_optional'));