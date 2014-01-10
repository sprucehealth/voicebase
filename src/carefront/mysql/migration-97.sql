start transaction;
insert into app_text (app_text_tag, comment) values ('txt_hypertension', 'hypertension');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Hypertension', (select id from app_text where app_text_tag='txt_hypertension'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_hypertension'), (select id from answer_type where atype='a_type_multiple_choice'),'a_other_condition_acne_hypertension',4,'ACTIVE');

insert into app_text (app_text_tag, comment) values ('txt_poly_ovary_syndrome', 'polycystic ovary syndrome');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Polycystic Ovary Syndrome', (select id from app_text where app_text_tag='txt_poly_ovary_syndrome'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_poly_ovary_syndrome'), (select id from answer_type where atype='a_type_multiple_choice'),'a_other_condition_acne_polycystic_ovary_syndrome',5,'ACTIVE');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_none_of_the_above'), (select id from answer_type where atype='a_type_multiple_choice_none'),'a_other_condition_acne_none',6,'ACTIVE');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_other_conditions_acne'), (select id from app_text where app_text_tag='txt_liver_disease_condition'), (select id from answer_type where atype='a_type_multiple_choice_none'),'a_other_condition_acne_liver_disease',7,'ACTIVE');


update potential_answer set ordering=8 where potential_answer_tag='a_other_condition_acne_colitis';
update potential_answer set ordering=9 where potential_answer_tag='a_other_condition_acne_gastiris';
update potential_answer set ordering=10 where potential_answer_tag='a_other_condition_acne_hypertension';
update potential_answer set ordering=11 where potential_answer_tag='a_other_condition_acne_kidney_condition';
update potential_answer set ordering=12 where potential_answer_tag='a_other_condition_acne_liver_disease';
update potential_answer set ordering=13 where potential_answer_tag='a_other_condition_acne_lupus';
update potential_answer set ordering=14 where potential_answer_tag='a_other_condition_acne_polycystic_ovary_syndrome';
update potential_answer set ordering=15 where potential_answer_tag='a_other_condition_acne_none';

commit;