start transaction;

update question set qtype_id=(select id from question_type where qtype='q_type_single_select') where question_tag='q_allergic_medications';
update question set qtext_app_text_id=NULL where question_tag='q_allergic_medication_entry';
update localized_text set ltext ='Add Medication' where app_text_id = (select app_text_id from question_fields where question_field='add_text' and question_id = (select id from question where question_tag='q_allergic_medication_entry'));
commit;