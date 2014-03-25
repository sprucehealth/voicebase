alter table treatment add patient_id int unsigned;
alter table treatment add foreign key (patient_id) references patient(id);
update treatment inner join treatment_plan 
	on treatment_plan_id = treatment_plan.id inner join patient_visit on patient_visit_id = treatment_plan.patient_visit_id
	set treatment.patient_id = patient_visit.patient_id
	where treatment.treatment_plan_id is not null;

create table dntf_mapping (
	id int unsigned not null auto_increment,
	treatment_id int unsigned not null,
	rx_refill_request_id int unsigned not null,
	foreign key (treatment_id) references treatment(id),
	foreign key (rx_refill_request_id) references rx_refill_request(id),
	primary key(id)
) character set utf8;