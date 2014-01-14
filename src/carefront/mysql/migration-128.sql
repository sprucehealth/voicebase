update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_severity_mild') where potential_answer_tag='a_doctor_acne_severity_mild';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_severity_moderate') where potential_answer_tag='a_doctor_acne_severity_moderate';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_acne_severity_severe') where potential_answer_tag='a_doctor_acne_severity_severity';
