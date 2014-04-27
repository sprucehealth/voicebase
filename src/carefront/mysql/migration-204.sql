set @en_id = (select id from languages_supported where language='en');
insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_allergic_medication_entry', 'empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_allergic_medication_entry');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'No medications specified', @app_text_id);

set @question_id = (select id from question where question_tag='q_allergic_medication_entry');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);



insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_current_medications_entry','empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_current_medications_entry');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'No medications specified', @app_text_id);

set @question_id = (select id from question where question_tag='q_current_medications_entry');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);


insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_list_prev_skin_condition_diagnosis','empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_list_prev_skin_condition_diagnosis');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'None', @app_text_id);

set @question_id = (select id from question where question_tag='q_list_prev_skin_condition_diagnosis');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);



insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_changes_acne_worse','empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_changes_acne_worse');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'Patient chose not to answer', @app_text_id);

set @question_id = (select id from question where question_tag='q_changes_acne_worse');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);



insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_acne_prev_treatment_list','empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_treatment_list');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'No prescriptipns specified', @app_text_id);

set @question_id = (select id from question where question_tag='q_acne_prev_treatment_list');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);


-- insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_acne_prev_otc_treatment_list','empty state text');

-- set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_acne_prev_otc_treatment_list');
-- insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'No OTC treatments specified', @app_text_id);

-- set @question_id = (select id from question where question_tag='q_acne_prev_otc_treatment_list');
-- insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);


insert into app_text(app_text_tag, comment) values ('txt_empty_state_q_anything_else_acne','empty state text');

set @app_text_id = (select id from app_text where app_text_tag='txt_empty_state_q_anything_else_acne');
insert into localized_text (language_id, ltext, app_text_id) values (@en_id,'Patient chose not to answer', @app_text_id);

set @question_id = (select id from question where question_tag='q_anything_else_acne');
insert into question_fields(question_field, question_id, app_text_id) values ('empty_state_text', @question_id, @app_text_id);
