alter table patient_visit add diagnosis text;
alter table health_condition add medicine_branch varchar(300) not null default "";
update health_condition set medicine_branch = 'Dermatology' where health_condition_tag = 'health_condition_acne';