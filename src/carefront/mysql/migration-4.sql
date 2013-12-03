-- Adding missing questions in the medical history section
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into app_text (app_text_tag, comment) values ('txt_allergy_topical_medication', 'txt for determining whether patient has been allergic to topical medication');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_allergy_topical_medication'), 'Have you ever had an allergic reaction to a topical medication?');

insert into app_text (app_text_tag, comment) values ('txt_summary_allergy_topical_medication', 'txt summary for determining whether patient has been allergic to topical medication');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_summary_allergy_topical_medication'), 'Topical Medication Allergies');


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_allergy_topical_medication'), (select id from app_text where app_text_tag='txt_summary_allergy_topical_medication'), 'q_topical_allergic_medications', 1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_topical_allergic_medications'), (select id from app_text where app_text_tag='txt_yes'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_topical_allergic_medication_yes', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_topical_allergic_medications'), (select id from app_text where app_text_tag='txt_no'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_topical_allergic_medication_no', 1);


insert into app_text (app_text_tag, comment) values ('txt_other_condition_acne', 'txt for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_other_condition_acne'), 'Do you currently have or have been treated for any of the following conditions?');

insert into app_text (app_text_tag, comment) values ('txt_summary_other_condition_acne', 'txt for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_summary_other_condition_acne'), 'Other conditions');

insert into app_text (app_text_tag, comment) values ('txt_gasitris', 'txt response for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_gasitris'), 'Gasitris');

insert into app_text (app_text_tag, comment) values ('txt_colitis', 'txt response for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_colitis'), 'Colitis');

insert into app_text (app_text_tag, comment) values ('txt_kidney_disease', 'txt response for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_kidney_disease'), 'Kidney Disease');

insert into app_text (app_text_tag, comment) values ('txt_lupus', 'txt response for determining any other conditions patient may have been diagnosed for in the past');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_lupus'), 'Lupus');

commit;
