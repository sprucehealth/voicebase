-- Adding data that corresponds to summary of a particular answer
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_not_effective', 'txt summary for treatment not effective');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_not_effective'), 'Not very effective');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_somewhat_effective', 'txt summary for treatment somewhat effective');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_somewhat_effective'), 'Somewhat effective');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_very_effective', 'txt summary for treatment very effective');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_very_effective'), 'Very effective');

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_not_effective') where potential_answer_tag = 'a_effective_treatment_not_very';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_somewhat_effective') where potential_answer_tag = 'a_effective_treatment_somewhat';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_very_effective') where potential_answer_tag = 'a_effective_treatment_very';

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_not_using', 'txt summary for not currently using treatment');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_not_using'), 'Not currently using it');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_using', 'txt summary for using current treatment');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_using'), 'Currently using it');

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_not_using') where potential_answer_tag = 'a_using_treatment_no';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_using') where potential_answer_tag = 'a_using_treatment_yes';


insert into app_text (app_text_tag, comment) values ('txt_answer_summary_less_month', 'txt summary for using treatment less than a month');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_less_month'), 'Used for less than one month');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_two_five_months', 'txt summary for using treatment 2-5 months');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_two_five_months'), 'Used for 2-5 months');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_six_eleven_months', 'txt summary for using treamtent 6-11 months');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_six_eleven_months'), 'Used for 6-11 months');

insert into app_text (app_text_tag, comment) values ('txt_answer_summary_twelve_plus_months', 'txt summary for using treatment 12+ months');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_answer_summary_twelve_plus_months'), 'Used for 12+ months');

update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_less_month') where potential_answer_tag = 'a_length_treatment_less_one';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_two_five_months') where potential_answer_tag = 'a_length_treatment_two_five_months';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_six_eleven_months') where potential_answer_tag = 'a_length_treatment_six_eleven_months';
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag = 'txt_answer_summary_twelve_plus_months') where potential_answer_tag = 'a_length_treatment_twelve_plus_months';

commit;