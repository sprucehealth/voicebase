alter table treatment_plan add column patient_id int unsigned;
update treatment_plan 
	inner join patient_visit on patient_visit_id = patient_visit.patient_id
	set treatment_plan.patient_id = patient_visit.patient_id;
alter table treatment_plan modify column patient_id int unsigned not null;
alter table treatment_plan add foreign key (patient_id) references patient(id);