start transaction;

insert into app_text (app_text_tag, comment) values ('txt_six_twelve_months_ago', '6-12 months ago');
insert into localized_text (language_id, ltext, app_text_id) values (1, "6-12 months ago", (select id from app_text where app_text_tag='txt_six_twelve_months_ago'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_onset_acne'), (select id from answer_type where atype='a_type_multiple_choice'),'a_six_twelve_months_ago',(select id from app_text where app_text_tag='txt_six_twelve_months_ago'),5,'ACTIVE');
update potential_answer set ordering = 4 where potential_answer_tag = 'a_onset_six_months';
update potential_answer set ordering = 6 where potential_answer_tag = 'a_onset_one_two_years';	
update potential_answer set ordering = 7 where potential_answer_tag = 'a_onset_more_two_years';
update localized_text set ltext = "2 or more years ago" where app_text_id = (select id from app_text where app_text_tag='txt_more_than_two_years');

update potential_answer set ordering = 5 where potential_answer_tag='a_discoloration';
update potential_answer set ordering = 6 where potential_answer_tag ='a_scarring';
update potential_answer set ordering = 7 where potential_answer_tag ='a_painful_touch';
update potential_answer set ordering = 8 where potential_answer_tag ='a_cysts';
update potential_answer set ordering = 9 where potential_answer_tag ='a_symptoms_none';

update localized_text set ltext = 'Ex: sports, new cosmetics, increased stress' where app_text_id = (select id from app_text where app_text_tag='txt_examples_changes_acne_worse');
update localized_text set ltext = 'List the acne treatments that you are currently using or have tried in the past.' where app_text_id = (select id from app_text where app_text_tag='txt_list_medications_acne');
update localized_text set ltext = "Optional: Is there anything else you’d like to share about your skin with Dr. %s?" where app_text_id = (select id from app_text where app_text_tag='txt_anything_else_acne');
update localized_text set ltext = 'Anything else you’d like your doctor to know?' where app_text_id = (select id from app_text where app_text_tag='txt_hint_anything_else_acne_treatment');

commit;