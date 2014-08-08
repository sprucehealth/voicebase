create table patient_pcp (
	patient_id int unsigned not null,
	physician_name varchar(500) not null,
	phone_number varchar(30) not null,
	practice_name varchar(300) not null,
	email varchar(300) not null,
	fax_number varchar(300) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (patient_id) references patient(id),
	primary key (patient_id)
) character set utf8;

create table patient_emergency_contact (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	full_name varchar(1024) not null,
	phone_number varchar(30) not null,
	relationship varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;