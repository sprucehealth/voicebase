start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into app_text (app_text_tag, comment) values ('txt_first_acne_experience', 'txt for when you first started experiencing acne');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_first_acne_experience'), 'When did you first begin experiencing acne?');

insert into app_text (app_text_tag, comment) values ('txt_during_puberty', 'txt response of during puberty');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_during_puberty'), 'During puberty');

insert into app_text (app_text_tag, comment) values ('txt_within_last_six_months', 'txt response of within last six months');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_within_last_six_months'), 'Within the last six months');

insert into app_text (app_text_tag, comment) values ('txt_one_two_years_ago', 'txt response of 1-2 years ago');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_one_two_years_ago'), '1-2 years ago');

insert into app_text (app_text_tag, comment) values ('txt_more_than_two_years', 'txt response of more than 2 years ago');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_more_than_two_years'), 'More than 2 years ago');

insert into app_text (app_text_tag, comment) values ('txt_onset_symptoms', 'txt summary for onset of symptoms');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_onset_symptoms'), 'Onset of symptoms');

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_first_acne_experience'), (select id from app_text where app_text_tag='txt_onset_symptoms'), 'q_onset_acne', 1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_onset_acne'), (select id from app_text where app_text_tag='txt_during_puberty'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_puberty', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_onset_acne'), (select id from app_text where app_text_tag='txt_within_last_six_months'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_onset_six_months', 1);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_onset_acne'), (select id from app_text where app_text_tag='txt_one_two_years_ago'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_onset_one_two_years', 2);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_onset_acne'), (select id from app_text where app_text_tag='txt_more_than_two_years'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_onset_more_two_years', 3);


insert into app_text (app_text_tag, comment) values ('txt_acne_symtpoms', 'txt for asking the user if they are experiencing acne symptoms');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_acne_symtpoms'), 'Are you experiencing any of the following symptoms with your acne?');

insert into app_text (app_text_tag, comment) values ('txt_painful_touch', 'txt for response of acne being painful to touch');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_painful_touch'), 'Painful to the touch');

insert into app_text (app_text_tag, comment) values ('txt_scarring', 'txt for response of acne being scarring');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_scarring'), 'Scarring');

insert into app_text (app_text_tag, comment) values ('txt_discoloration', 'txt for response of acne causing discoloration');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_discoloration'), 'Discoloration');

insert into app_text (app_text_tag, comment) values ('txt_additional_symptoms', 'txt for summarizing additional symptoms');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_additional_symptoms'), 'Additional Symptoms');


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_multiple_choice'), (select id from app_text where app_text_tag='txt_acne_symtpoms'), (select id from app_text where app_text_tag='txt_additional_symptoms'), 'q_acne_symptoms', 1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_acne_symptoms'), (select id from app_text where app_text_tag='txt_painful_touch'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_painful_touch', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_acne_symptoms'), (select id from app_text where app_text_tag='txt_scarring'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_scarring', 1);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_acne_symptoms'), (select id from app_text where app_text_tag='txt_discoloration'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_discoloration', 2);


insert into app_text (app_text_tag, comment) values ('txt_acne_worse_period', 'txt for asking female patients if their acne gets worse with periods');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_acne_worse_period'), 'Does your acne get worse with your period?');

insert into app_text (app_text_tag, comment) values ('txt_periods_regular', 'txt for asking female patients if their periods are regular');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_periods_regular'), 'Are your periods regular?');

insert into app_text (app_text_tag, comment) values ('txt_menstrual_cycle', 'txt for summarizing information about txt_menstrual_cycle');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_menstrual_cycle'), 'Menstrual cycle');


insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_acne_worse_period'), (select id from app_text where app_text_tag='txt_menstrual_cycle'), 'q_acne_worse_period', 0);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_acne_worse_period'), (select id from app_text where app_text_tag='txt_yes'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_worse_yes', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_acne_worse_period'), (select id from app_text where app_text_tag='txt_no'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_worse_no', 1);

set @parent_question_id = (select id from question where question_tag='q_acne_worse_period');
insert into question (qtype_id, qtext_app_text_id, question_tag, parent_question_id, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_periods_regular'), 'q_periods_regular', @parent_question_id, 0);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_periods_regular'), (select id from app_text where app_text_tag='txt_yes'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_periods_regular_yes', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_periods_regular'), (select id from app_text where app_text_tag='txt_no'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_periods_regular_no', 1);


insert into app_text (app_text_tag, comment) values ('txt_skin_description', 'txt for question to descibe skin');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_skin_description'), 'How would you describe your skin?');

insert into app_text (app_text_tag, comment) values ('txt_normal_skin', 'txt for response to skin description as normal');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_normal_skin'), 'Normal');

insert into app_text (app_text_tag, comment) values ('txt_oily_skin', 'txt for response to skin description as oily');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_oily_skin'), 'Oily');

insert into app_text (app_text_tag, comment) values ('txt_dry_skin', 'txt for response to skin description as dry');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_dry_skin'), 'Dry');

insert into app_text (app_text_tag, comment) values ('txt_combination_skin', 'txt for response to skin description as combination');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_combination_skin'), 'Combination');

insert into app_text (app_text_tag, comment) values ('txt_skin_type', 'txt for summarizing skin type');
insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_skin_type'), 'Skin type');

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'), (select id from app_text where app_text_tag='txt_skin_description'), (select id from app_text where app_text_tag='txt_skin_type'), 'q_skin_description', 1);

insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_skin_description'), (select id from app_text where app_text_tag='txt_normal_skin'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_normal_skin', 0);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_skin_description'), (select id from app_text where app_text_tag='txt_oily_skin'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_oil_skin', 1);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_skin_description'), (select id from app_text where app_text_tag='txt_dry_skin'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_dry_skin', 2);
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_skin_description'), (select id from app_text where app_text_tag='txt_combination_skin'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_combination_skin', 3);

commit;



