create table patient_visit_message (
	patient_visit_id int unsigned not null,
	message text not null,
	creation_timestamp timestamp not null default current_timestamp,
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (patient_visit_id)
) character set utf8;
