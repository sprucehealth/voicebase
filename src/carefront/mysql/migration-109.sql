start transaction;
insert into app_text (app_text_tag, comment) values ('txt_summary_periods_regular', 'regular periods summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Regular periods", (select id from app_text where app_text_tag='txt_summary_periods_regular'));
update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_summary_periods_regular') where question_tag='q_periods_regular';

insert into app_text (app_text_tag, comment) values ('txt_summary_periods_worse', 'worse periods summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Worse with period", (select id from app_text where app_text_tag='txt_summary_periods_worse'));
update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_summary_periods_worse') where question_tag='q_acne_worse_period';

update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_short_acne_worse') where question_tag='q_acne_worse';

insert into app_text (app_text_tag, comment) values ('txt_summary_environment_factors', 'potential environment factors');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Potential environment factors", (select id from app_text where app_text_tag='txt_summary_environment_factors'));
update question set qtext_short_text_id=(select id from app_text where app_text_tag='txt_summary_environment_factors') where question_tag='q_changes_acne_worse';

update localized_text set ltext='Types of symptoms' where app_text_id = (select qtext_short_text_id from question where question_tag='q_acne_symptoms');
update localized_text set ltext='Onset of symptoms' where app_text_id = (select qtext_short_text_id from question where question_tag='q_onset_acne');

commit;

