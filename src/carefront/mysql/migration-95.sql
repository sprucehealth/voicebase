start transaction;
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_one_or_less') where potential_answer_tag='a_length_current_medication_less_than_month';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_two_five_months') where potential_answer_tag='a_length_current_medication_two_five_months';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_six_eleven_months') where potential_answer_tag='a_length_current_medication_six_eleven_months';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_twelve_plus_months') where potential_answer_tag='a_length_current_medication_twelve_plus_months';

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_less_one_month') where potential_answer_tag='a_length_current_medication_less_than_month';

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_two_five_months') where potential_answer_tag='a_length_current_medication_two_five_months';

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_six_eleven_months') where potential_answer_tag='a_length_current_medication_six_eleven_months';

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_twelve_plus_months') where potential_answer_tag='a_length_current_medication_twelve_plus_months';

commit;