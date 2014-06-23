create table patient_case (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	status varchar(100) not null,
	health_condition_id int unsigned not null,
	foreign key (patient_id) references patient(id),
	foreign key (health_condition_id) references health_condition(id),
	primary key (id)
) character set utf8;

create table patient_case_care_provider_assignment (
  id int unsigned not null auto_increment,
  patient_case_id int unsigned not null,
  provider_id int unsigned not null,
  role_type_id int unsigned not null,
  creation_date timestamp(6) not null default current_timestamp(6),
  status varchar(100) not null,
  foreign key (patient_case_id) references patient_case(id),
  foreign key (role_type_id) references role_type(id),
  primary key (id),
  unique key (patient_case_id, provider_id, role_type_id)
) character set utf8;

-- Tie the patient visit to the case
alter table patient_visit add column patient_case_id int unsigned;
insert into patient_case (patient_id, status, health_condition_id) select patient_id,'OPEN', (select id from health_condition where health_condition_tag='health_condition_acne') from patient_visit;
update patient_visit 
	inner join patient_case on patient_case.patient_id = patient_visit.patient_id
	set patient_case_id = patient_case.id ;


-- Tie the treatment plan to the case
alter table treatment_plan add column patient_case_id int unsigned;
update treatment_plan 
		inner join treatment_plan_patient_visit_mapping on treatment_plan_id = treatment_plan.id
		inner join patient_visit on patient_visit.id = treatment_plan_patient_visit_mapping.patient_visit_id
	set treatment_plan.patient_case_id = patient_visit.patient_case_id;

-- Give full access to the case to all doctors currently assigned to handle a patient visit
insert into patient_case_care_provider_assignment (patient_case_id, provider_id, role_type_id, status)
	select patient_case_id, provider_id, role_type_id, patient_visit_care_provider_assignment.status
		from patient_visit_care_provider_assignment
		inner join patient_visit on patient_visit_care_provider_assignment.patient_visit_id = patient_visit.id
		where patient_visit_care_provider_assignment.status != "INACTIVE";



