-- Adding questions for doctor to diagnose
start transaction;


insert into app_text (app_text_tag, comment) values ('txt_what_diagnosis', 'what is your diagnosisa');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_what_diagnosis'), 1, "What's your diagnosis?");

insert into app_text (app_text_tag, comment) values ('txt_acne_vulgaris', 'acne vulgaris');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_vulgaris'), 1, "Acne Vulgaris");


insert into app_text (app_text_tag, comment) values ('txt_acne_rosacea', 'acne rosacea');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_rosacea'), 1, "Acne Rosacea");


insert into question (qtype_id,qtext_app_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'),(select id from app_text where app_text_tag='txt_what_diagnosis'), 'q_acne_diagnosis', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_diagnosis'),(select id from app_text where app_text_tag='txt_acne_vulgaris'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_vulgaris', 0);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_diagnosis'),(select id from app_text where app_text_tag='txt_acne_rosacea'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_rosacea', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_diagnosis'),(select id from app_text where app_text_tag='txt_something_else_visit_reason'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_something_else', 2);


insert into app_text (app_text_tag, comment) values ('txt_acne_severity', 'how severe is the patients acne');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_severity'), 1, "How severe is the patient's acne?");

insert into app_text (app_text_tag, comment) values ('txt_acne_severity_mild', 'acne severity mild');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_severity_mild'), 1, "Mild");

insert into app_text (app_text_tag, comment) values ('txt_acne_severity_moderate', 'acne severity moderate');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_severity_moderate'), 1, "Moderate");

insert into app_text (app_text_tag, comment) values ('txt_acne_severity_severe', 'acne severity severe');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_severity_severe'), 1, "Severe");

insert into question (qtype_id,qtext_app_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_select'),(select id from app_text where app_text_tag='txt_acne_severity'), 'q_acne_severity', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_severity'),(select id from app_text where app_text_tag='txt_acne_severity_mild'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_severity_mild', 0);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_severity'),(select id from app_text where app_text_tag='txt_acne_severity_moderate'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_severity_moderate', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_severity'),(select id from app_text where app_text_tag='txt_acne_severity_severe'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_doctor_acne_severity_severity', 2);


insert into app_text (app_text_tag, comment) values ('txt_acne_type', 'type of acne');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_type'), 1, "What type of acne do they have?");

insert into app_text (app_text_tag, comment) values ('txt_acne_whiteheads', 'acne whiteheads');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_whiteheads'), 1, "Whiteheads");

insert into app_text (app_text_tag, comment) values ('txt_acne_pustules', 'acne pustules');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_pustules'), 1, "Pustules");

insert into app_text (app_text_tag, comment) values ('txt_acne_nodules', 'acne nodules');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_nodules'), 1, "Nodules");

	
insert into app_text (app_text_tag, comment) values ('txt_acne_inflammatory', 'acne inflammatory');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_inflammatory'), 1, "Inflammatory");


insert into app_text (app_text_tag, comment) values ('txt_acne_blackheads', 'acne blackheads');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_blackheads'), 1, "Blackheads");

insert into app_text (app_text_tag, comment) values ('txt_acne_papules', 'acne papules');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_papules'), 1, "Papules");

insert into app_text (app_text_tag, comment) values ('txt_acne_cysts', 'acne cysts');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_cysts'), 1, "Cysts");

insert into app_text (app_text_tag, comment) values ('txt_acne_hormonal', 'acne hormonal');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_acne_hormonal'), 1, "Hormonal");


insert into question (qtype_id,qtext_app_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_multiple_choice'),(select id from app_text where app_text_tag='txt_acne_type'), 'q_acne_type', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_whiteheads'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_whiteheads', 0);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_pustules'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_pustules', 1);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_nodules'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_nodules', 2);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_inflammatory'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_inflammatory', 3);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_blackheads'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_blackheads', 4);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_papules'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_papules', 5);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_cysts'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_cysts', 6);
insert into potential_answer (question_id,answer_localized_text_id, atype_id,potential_answer_tag,ordering) values ((select id from question where question_tag='q_acne_type'),(select id from app_text where app_text_tag='txt_acne_hormonal'), (select id from answer_type where atype='a_type_multiple_choice'), 'a_acne_hormonal', 7);


commit;