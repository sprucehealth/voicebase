drop table dr_diagnosis_intake;

alter table patient_info_intake add column role varchar(100) not null;
alter table patient_info_intake drop foreign key patient_info_intake_ibfk_5;
alter table patient_info_intake change column patient_id role_id int unsigned not null;
