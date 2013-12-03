-- Adding new question type to capture list of topical medication allergies
start transaction;

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required) values ((select id from question_type where qtype='q_type_compound'), (select id from app_text where app_text_tag='txt_add_medicataion'), (select id from app_text where app_text_tag='txt_summary_allergy_topical_medication'), 'q_topical_allergies_medication_entry', 0);
insert into potential_answer (question_id, atype_id, potential_answer_tag, ordering) values ((select id from question where question_tag='q_topical_allergies_medication_entry'), (select id from answer_type where atype='a_type_autocomplete_entry'), 'a_topical_medication_allergy_entry', 0);

commit;