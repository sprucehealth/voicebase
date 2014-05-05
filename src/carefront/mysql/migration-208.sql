update patient_visit_follow_up inner join treatment_plan on patient_visit_follow_up.patient_visit_id = treatment_plan.patient_visit_id 
	set treatment_plan_id = treatment_plan.id where (patient_visit_follow_up.patient_visit_id is not null) and (treatment_plan_id is null); 
delete from patient_visit_follow_up where treatment_plan_id is null;

update regimen inner join treatment_plan on regimen.patient_visit_id = treatment_plan.patient_visit_id 
	set treatment_plan_id = treatment_plan.id where (regimen.patient_visit_id is not null) and (treatment_plan_id is null); 
delete from regimen where treatment_plan_id is null;

update advice inner join treatment_plan on advice.patient_visit_id = treatment_plan.patient_visit_id 
	set treatment_plan_id = treatment_plan.id where (advice.patient_visit_id is not null) and (treatment_plan_id is null); 
delete from advice where treatment_plan_id is null;

update diagnosis_summary inner join treatment_plan on diagnosis_summary.patient_visit_id = treatment_plan.patient_visit_id 
	set treatment_plan_id = treatment_plan.id where (diagnosis_summary.patient_visit_id is not null) and (treatment_plan_id is null); 
delete from diagnosis_summary where treatment_plan_id is null;

alter table patient_visit_follow_up drop foreign key patient_visit_follow_up_ibfk_1;
alter table patient_visit_follow_up drop column patient_visit_id;
alter table patient_visit_follow_up drop foreign key patient_visit_follow_up_ibfk_3;
alter table patient_visit_follow_up modify column treatment_plan_id int unsigned not null;
alter table patient_visit_follow_up add foreign key (treatment_plan_id) references treatment_plan(id);

alter table regimen drop foreign key regimen_ibfk_1;
alter table regimen drop column patient_visit_id;
alter table regimen drop foreign key regimen_ibfk_3;
alter table regimen modify column treatment_plan_id int unsigned not null;
alter table regimen add foreign key (treatment_plan_id) references treatment_plan(id);

alter table advice drop foreign key advice_ibfk_1;
alter table advice drop column patient_visit_id;
alter table advice drop foreign key advice_ibfk_3;
alter table advice modify column treatment_plan_id int unsigned not null;
alter table advice add foreign key (treatment_plan_id) references treatment_plan(id);

alter table diagnosis_summary drop foreign key diagnosis_summary_ibfk_1;
alter table diagnosis_summary drop column patient_visit_id;
alter table diagnosis_summary drop foreign key diagnosis_summary_ibfk_3;
alter table diagnosis_summary modify column treatment_plan_id int unsigned not null;
alter table diagnosis_summary add foreign key (treatment_plan_id) references treatment_plan(id);