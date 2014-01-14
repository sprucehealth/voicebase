start transaction;

update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_perioral_dermitits') where potential_answer_tag = 'a_doctor_acne_perioral_dermatitis';
update localized_text set ltext='Acne vulgaris' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag ='a_doctor_acne_vulgaris');
update localized_text set ltext='Acne rosacea' where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag ='a_doctor_acne_rosacea');

commit;