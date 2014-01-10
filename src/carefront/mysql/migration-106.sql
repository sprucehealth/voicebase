drop table patient_phone;
create table patient_phone (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	phone varchar(100) not null,
	phone_type varchar(100) not null,
	status varchar(100) not null,
	foreign key (patient_id) references patient(id),
	primary key(id)
) character set utf8;

