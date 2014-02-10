
alter table info_intake drop foreign key info_intake_ibfk_1;
alter table info_intake change patient_visit_id context_id int unsigned not null;

alter table diagnosis_summary drop foreign key diagnosis_summary_ibfk_1;
alter table diagnosis_summary drop column patient_visit_id;
alter table diagnosis_summary add column treatment_plan_id int unsigned;
alter table diagnosis_summary add foreign key (treatment_plan_id) references treatment_plan(id);

alter table regimen drop foreign key regimen_ibfk_1;
alter table regimen drop column patient_visit_id;
alter table regimen add column treatment_plan_id int unsigned;
alter table regimen add foreign key (treatment_plan_id) references treatment_plan(id);

alter table advice drop foreign key advice_ibfk_1;
alter table advice drop column patient_visit_id;
alter table advice add column treatment_plan_id int unsigned;
alter table advice add foreign key (treatment_plan_id) references treatment_plan(id);

alter table patient_visit_follow_up drop foreign key patient_visit_follow_up_ibfk_1;
alter table patient_visit_follow_up drop column patient_visit_id;
alter table patient_visit_follow_up add column treatment_plan_id int unsigned;
alter table patient_visit_follow_up add foreign key (treatment_plan_id) references treatment_plan(id);
