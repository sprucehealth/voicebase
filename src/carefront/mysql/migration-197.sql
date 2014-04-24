alter table treatment modify column dispense_value decimal(11,6) not null;
alter table pharmacy_dispensed_treatment modify column dispense_value decimal(11,6) not null;
alter table requested_treatment modify column dispense_value decimal(11,6) not null;
alter table unlinked_dntf_treatment modify column dispense_value decimal(11,6) not null;