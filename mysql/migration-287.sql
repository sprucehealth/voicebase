alter table pharmacy_dispensed_treatment modify column refills int not null;
update requested_treatment set erx_last_filled_date = NULL where erx_last_filled_date='0000-00-00 00:00:00';
alter table requested_treatment modify column refills int not null; 
delete from dispense_unit where dispense_unit_text_id = (select app_text_id from localized_text where ltext = 'Mutually Defined'); 
