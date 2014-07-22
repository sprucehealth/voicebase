create table patient_alerts (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	alert text not null,
	source varchar(100) not null,
	source_id int unsigned not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;