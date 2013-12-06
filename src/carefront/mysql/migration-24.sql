-- drop the opened_date column in the patient visit
alter table patient_visit drop opened_date;

alter table patient_visit add submitted_date timestamp on update current_timestamp;