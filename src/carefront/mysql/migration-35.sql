start transaction;
delete from patient_visit_provider_assignment;
delete from patient_visit_provider_group;
delete from care_provider_state_elligibility;
delete from provider_role where provider_tag='PRIMARY_DOCTOR';
commit;

alter table patient_visit_provider_assignment drop foreign key patient_visit_provider_assignment_ibfk_1;
alter table patient_visit_provider_group drop foreign key patient_visit_provider_group_ibfk_1;
alter table patient_visit_provider_assignment drop column patient_visit_id;
alter table patient_visit_provider_group drop column patient_visit_id;

alter table patient_visit_provider_assignment add patient_id int unsigned not null;
alter table patient_visit_provider_assignment add foreign key (patient_id) references patient(id);

alter table patient_visit_provider_group add patient_id int unsigned not null;
alter table patient_visit_provider_group add foreign key (patient_id) references patient(id);
