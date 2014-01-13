create table patient_visit_event (
	id int unsigned not null,
	patient_visit_id int unsigned,
	event varchar(100) not null,
	status varchar(100) not null,
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (id)
) character set utf8;