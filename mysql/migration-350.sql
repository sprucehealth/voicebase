alter table patient_alerts add column status varchar(32);
update patient_alerts set status='ACTIVE';
alter table patient_alerts modify column status varchar(32) not null;

insert into question (qtype_id, qtext_app_text_id, qtext_short_text_id, question_tag, required, to_alert, alert_app_text_id) 
	SELECT qtype_id, qtext_app_text_id, qtext_short_text_id, 'q_medication_allergies_since_visit_entry', required, to_alert, alert_app_text_id
	FROM question where question_tag='q_allergic_medication_entry';

insert into question_fields (question_field, question_id, app_text_id)
	SELECT question_field, (select id from question where question_tag='q_medication_allergies_since_visit_entry'), app_text_id
	FROM question_fields where question_id = (select id from question where question_tag='q_allergic_medication_entry');