alter table requested_treatment add column doctor_id int unsigned;
alter table requested_treatment add foreign key (doctor_id) references doctor(id);

alter table pharmacy_dispensed_treatment add column doctor_id int unsigned;
alter table pharmacy_dispensed_treatment add foreign key (doctor_id) references doctor(id);