alter table treatment modify column dispense_value decimal(6,3) not null;
alter table pharmacy_dispensed_treatment modify column dispense_value decimal(6,3) not null;
alter table requested_treatment modify column dispense_value decimal(6,3) not null;
alter table unlinked_dntf_treatment modify column dispense_value decimal(6,3) not null;