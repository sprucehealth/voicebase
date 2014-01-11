start transaction;

update localized_text set ltext='Pregnancy' where app_text_id = (select qtext_short_text_id from question where question_tag='q_pregnancy_planning');

insert into app_text (app_text_tag, comment) values ('txt_summary_is_pregnant', 'is pregnant summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Currently pregnant", (select id from app_text where app_text_tag='txt_summary_is_pregnant'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_summary_is_pregnant') where potential_answer_tag='a_pregnant';

insert into app_text (app_text_tag, comment) values ('txt_summary_is_nursing', 'is nursing summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Currently nursing", (select id from app_text where app_text_tag='txt_summary_is_nursing'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_summary_is_nursing') where potential_answer_tag='a_nursing';

insert into app_text (app_text_tag, comment) values ('txt_summary_planning_pregnancy', 'is planning pregnancy summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Currently planning a pregnancy", (select id from app_text where app_text_tag='txt_summary_planning_pregnancy'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_summary_planning_pregnancy') where potential_answer_tag='a_planning_pregnancy';

insert into app_text (app_text_tag, comment) values ('txt_summary_not_pregant_planning_nursing', 'is not pregnant planning or nursing summary');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Not currently pregnant, planning a pregnancy or nursing", (select id from app_text where app_text_tag='txt_summary_not_pregant_planning_nursing'));
update potential_answer set answer_summary_text_id = (select id from app_text where app_text_tag='txt_summary_not_pregant_planning_nursing') where potential_answer_tag='a_planning_pregnancy_none';

commit;	