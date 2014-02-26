alter table pharmacy_dispensed_treatment drop foreign key pharmacy_dispensed_treatment_ibfk_5;
alter table pharmacy_dispensed_treatment drop column dispense_unit_id;
alter table pharmacy_dispensed_treatment add column dispense_unit varchar(100) not null;


alter table unlinked_requested_treatment drop foreign key unlinked_requested_treatment_ibfk_4;
alter table unlinked_requested_treatment drop column dispense_unit_id;
alter table unlinked_requested_treatment add column dispense_unit varchar(100) not null;


