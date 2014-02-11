alter table info_intake drop foreign key info_intake_ibfk_1;
alter table info_intake change patient_visit_id context_id int unsigned not null;

alter table diagnosis_summary modify column patient_visit_id int unsigned;
alter table diagnosis_summary add column treatment_plan_id int unsigned;
alter table diagnosis_summary add foreign key (treatment_plan_id) references treatment_plan(id);

alter table regimen modify column patient_visit_id int unsigned;
alter table regimen add column treatment_plan_id int unsigned;
alter table regimen add foreign key (treatment_plan_id) references treatment_plan(id);

alter table advice modify column patient_visit_id int unsigned;
alter table advice add column treatment_plan_id int unsigned;
alter table advice add foreign key (treatment_plan_id) references treatment_plan(id);

alter table patient_visit_follow_up modify column patient_visit_id int unsigned;
alter table patient_visit_follow_up add column treatment_plan_id int unsigned;
alter table patient_visit_follow_up add foreign key (treatment_plan_id) references treatment_plan(id);
