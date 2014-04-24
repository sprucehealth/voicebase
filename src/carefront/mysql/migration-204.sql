alter table potential_answer add column to_alert tinyint(1);
alter table question add column to_alert tinyint(1);
alter table question add alert_app_text_id int unsigned;
alter table question add foreign key (alert_app_text_id) references app_text(id);


set @en_id = (select id from languages_supported where language='en');
insert into app_text(app_text_tag, comment) values ('txt_alert_q_pregnancy_planning', 'alert text');

set @app_text_id = (select id from app_text where app_text_tag='txt_alert_q_pregnancy_planning');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'Currently %s', @app_text_id);

set @question_id = (select id from question where question_tag='q_pregnancy_planning');
update question set to_alert = 1, alert_app_text_id=@app_text_id where id=@question_id;

update potential_answer set to_alert=1 where potential_answer_tag in ('a_planning_pregnancy', 'a_nursing', 'a_pregnant');


set @en_id = (select id from languages_supported where language='en');
insert into app_text(app_text_tag, comment) values ('txt_alert_q_allergic_medication_entry', 'alert text');

set @app_text_id = (select id from app_text where app_text_tag='txt_alert_q_allergic_medication_entry');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'Allergic to %s', @app_text_id);

set @question_id = (select id from question where question_tag='q_allergic_medication_entry');
update question set to_alert = 1, alert_app_text_id=@app_text_id where id=@question_id;
