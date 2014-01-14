create table doctor_phone (
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	phone varchar(100) not null,
	phone_type varchar(100) not null,
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) character set utf8;