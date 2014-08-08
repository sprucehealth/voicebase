alter table doctor_queue modify column enqueue_date timestamp not null default current_timestamp;
alter table unclaimed_case_queue modify column enqueue_date timestamp not null default current_timestamp;
