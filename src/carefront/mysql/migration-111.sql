start transaction;
insert into app_text (app_text_tag, comment) values ('txt_summary_other_past_skin_condition', 'summary for other past skin condition');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Other skin condition specified", (select id from app_text where app_text_tag='txt_summary_other_past_skin_condition'));
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_summary_other_past_skin_condition') where question_tag='q_other_skin_condition_entry';

commit;