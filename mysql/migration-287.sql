alter table pharmacy_dispensed_treatment modify column refills int not null;
alter table requested_treatment modify column refills int not null; 
delete from dispense_unit where dispense_unit_text_id = (select app_text_id from localized_text where ltext = 'Mutually Defined'); 
