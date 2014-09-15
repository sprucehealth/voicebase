update requested_treatment set status = 'CREATED';
update pharmacy_dispensed_treatment set status = 'CREATED';
update unlinked_dntf_treatment set status = 'CREATED'; 
update dr_favorite_treatment set status = 'CREATED' where status='ACTIVE';
update dr_treatment_template set status = 'CREATED' where status='ACTIVE';
