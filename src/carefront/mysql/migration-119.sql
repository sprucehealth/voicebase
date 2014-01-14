start transaction;
insert into app_text (app_text_tag, comment) values ('txt_perioral_dermitits', 'perioral dermatitis');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Perioral dermatitis", (select id from app_text where app_text_tag='txt_perioral_dermitits'));

update potential_answer set ordering=3 where potential_answer_tag='a_doctor_acne_vulgaris';
update potential_answer set ordering=4 where potential_answer_tag='a_doctor_acne_rosacea';
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_diagnosis'), (select id from answer_type where atype='a_type_multiple_choice'),'a_doctor_acne_perioral_dermatitis',(select id from app_text where app_text_tag='txt_perioral_dermitits'),5,'ACTIVE');
update potential_answer set ordering=6 where potential_answer_tag='a_doctor_acne_something_else';
update localized_text set ltext='Acne vulgaris' where app_text_id = (select answer_summary_text_id from potential_answer where potential_answer_tag ='a_doctor_acne_vulgaris');
update localized_text set ltext='Acne rosacea' where app_text_id = (select answer_summary_text_id from potential_answer where potential_answer_tag ='a_doctor_acne_rosacea');

insert into app_text (app_text_tag, comment) values ('txt_comedonal', 'comedonal');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Comedonal", (select id from app_text where app_text_tag='txt_comedonal'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_type'), (select id from answer_type where atype='a_type_multiple_choice'),'a_acne_comedonal',(select id from app_text where app_text_tag='txt_comedonal'),8,'ACTIVE');
update localized_text set ltext='Cystic' where app_text_id = (select answer_summary_text_id from potential_answer where potential_answer_tag ='a_acne_cysts');
update potential_answer set status='INACTIVE' where potential_answer_tag in ('a_acne_whiteheads','a_acne_nodules','a_acne_blackheads','a_acne_papules','a_acne_pustules');
update potential_answer set ordering = 9 where potential_answer_tag='a_acne_inflammatory';
update potential_answer set ordering = 10 where potential_answer_tag='a_acne_cysts';
update potential_answer set ordering = 11 where potential_answer_tag='a_acne_hormonal';	

insert into question (qtype_id, qtext_app_text_id,subtext_app_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_single_entry'), (select id from app_text where app_text_tag='txt_acne_type'), (select id from app_text where app_text_tag='txt_select_all_apply'), 'q_acne_rosacea_type', 1);

insert into app_text (app_text_tag, comment) values ('txt_erythematotelangiectatic_rosacea', 'Erythematotelangiectatic Rosacea');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Erythematotelangiectatic Rosacea", (select id from app_text where app_text_tag='txt_erythematotelangiectatic_rosacea'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_rosacea_type'), (select id from answer_type where atype='a_type_multiple_choice'),'a_acne_erythematotelangiectatic_rosacea',(select id from app_text where app_text_tag='txt_erythematotelangiectatic_rosacea'),0,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_papulopstular_rosacea', 'Papulopustular Rosacea');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Papulopustular Rosacea", (select id from app_text where app_text_tag='txt_papulopstular_rosacea'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_rosacea_type'), (select id from answer_type where atype='a_type_multiple_choice'),'a_acne_papulopstular_rosacea',(select id from app_text where app_text_tag='txt_papulopstular_rosacea'),1,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_rhinophyma_rosacea', 'Rhinophyma');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Rhinophyma", (select id from app_text where app_text_tag='txt_rhinophyma_rosacea'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_rosacea_type'), (select id from answer_type where atype='a_type_multiple_choice'),'a_acne_rhinophyma_rosacea',(select id from app_text where app_text_tag='txt_rhinophyma_rosacea'),2,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_ocular_rosacea', 'Ocular Rosacea');
insert into localized_text (language_id, ltext, app_text_id) values (1, "Ocular Rosacea", (select id from app_text where app_text_tag='txt_ocular_rosacea'));
insert into potential_answer (question_id, atype_id, potential_answer_tag,answer_summary_text_id, ordering, status) values ((select id from question where question_tag='q_acne_rosacea_type'), (select id from answer_type where atype='a_type_multiple_choice'),'a_acne_ocular_rosacea',(select id from app_text where app_text_tag='txt_ocular_rosacea'),3,'ACTIVE');

commit;