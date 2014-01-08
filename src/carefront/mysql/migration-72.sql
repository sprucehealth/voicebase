create table patient_visit_care_provider_assignment (
	id int unsigned not null,
	patient_visit_id int unsigned not null,
	provider_role int unsigned not null,
	provider_id int unsigned not null,
	status varchar(100) not null,
	assignment_date timestamp not null default current_timestamp,
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (id)
) character set utf8; 	