alter table pharmacy_dispensed_treatment modify column erx_id int unsigned not null;
alter table pharmacy_dispensed_treatment drop foreign key pharmacy_dispensed_treatment_ibfk_1;
alter table pharmacy_dispensed_treatment modify column treatment_id int unsigned;
alter table pharmacy_dispensed_treatment add column unlinked_requested_treatment_id int unsigned;
alter table pharmacy_dispensed_treatment add foreign key (unlinked_requested_treatment_id) references unlinked_requested_treatment(id);