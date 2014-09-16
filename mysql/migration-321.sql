update requested_treatment set status = 'CREATED';
update pharmacy_dispensed_treatment set status = 'CREATED';
update unlinked_dntf_treatment set status = 'CREATED'; 
update dr_favorite_treatment set status = 'CREATED' where status='ACTIVE';
update dr_treatment_template set status = 'CREATED' where status='ACTIVE';
alter table doctor_transaction add unique key (doctor_id, item_type, item_id);
