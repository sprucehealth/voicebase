alter table patient_visit_follow_up add column follow_up_date date not null;
alter table patient_visit_follow_up add column follow_up_value int unsigned not null;
alter table patient_visit_follow_up add column follow_up_unit varchar(100) not null;