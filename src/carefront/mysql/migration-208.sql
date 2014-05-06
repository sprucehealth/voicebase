alter table diagnosis_summary add column creation_date timestamp not null default current_timestamp on update current_timestamp;
alter table diagnosis_summary add unique key (treatment_plan_id);
alter table diagnosis_summary add column updated_by_doctor tinyint not null default 0;
