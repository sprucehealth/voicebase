alter table patient_case_message add column private tinyint(1) not null default 0;
alter table patient_case_message add column event_text text not null;
