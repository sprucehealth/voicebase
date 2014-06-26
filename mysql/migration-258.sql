alter table patient_care_provider_assignment add column creation_date timestamp(6) not null default current_timestamp(6);
alter table patient_care_provider_assignment drop foreign key patient_care_provider_assignment_ibfk_2;
alter table patient_care_provider_assignment drop column assignment_group_id;
drop table patient_care_provider_group;
drop table patient_visit_care_provider_assignment;

