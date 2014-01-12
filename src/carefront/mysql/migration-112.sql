create table patient_pharmacy_selection (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	pharmacy_id int unsigned not null,
	source varchar(100) not null,
	status varchar(100) not null,
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;