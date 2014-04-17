create table doctor_address_selection(
	id int unsigned not null auto_increment,
	address_id int unsigned not null,
	doctor_id int unsigned not null,
	foreign key (doctor_id) references doctor(id),
	foreign key (address_id) references address(id),
	primary key (id)
) character set utf8;