start transaction;
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_one_or_less') where potential_answer_tag='a_length_current_medication_less_than_month';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_two_five_months') where potential_answer_tag='a_length_current_medication_less_than_month';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_six_eleven_months') where potential_answer_tag='a_length_current_medication_less_than_month';
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag='txt_twelve_plus_months') where potential_answer_tag='a_length_current_medication_less_than_month';

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_taken_less_one_month', 'option to indicate that the patient has taken medication for less than one month');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Taken for less than 1 month', (select id from app_text where app_text_tag='txt_answer_summary_taken_less_one_month'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_less_one_month') where potential_answer_tag='a_length_current_medication_less_than_month';

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_taken_two_five_months', 'option to indicate that the patient has taken medication for 2-5 months');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Taken for 2-5 months', (select id from app_text where app_text_tag='txt_answer_summary_taken_two_five_months'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_two_five_months') where potential_answer_tag='a_length_current_medication_two_five_months';

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_taken_six_eleven_months', 'option to indicate that the patient has taken medication for 6-11 months');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Taken for 6-11 months', (select id from app_text where app_text_tag='txt_answer_summary_taken_six_eleven_months'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_six_eleven_months') where potential_answer_tag='a_length_current_medication_six_eleven_months';

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_taken_twelve_plus_months', 'option to indicate that the patient has taken medication for 12+ months');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Taken for 12+ months', (select id from app_text where app_text_tag='txt_answer_summary_taken_twelve_plus_months'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_answer_summary_taken_twelve_plus_months') where potential_answer_tag='a_length_current_medication_twelve_plus_months';

commit;