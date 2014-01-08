create table diagnosis_summary (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null, 
	doctor_id int unsigned not null,
	summary varchar(600) not null,
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) character set utf8;