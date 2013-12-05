start transaction;
insert into question_type (qtype) values ('q_type_autocomplete');
update question set qtype_id = (select id from question_type where qtype='q_type_autocomplete') where question_tag in ('q_acne_prev_treatment_list', 'q_allergic_medication_entry', 'q_current_medications_entry', 'q_topical_allergies_medication_entry');
commit;