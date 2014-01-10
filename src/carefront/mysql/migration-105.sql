drop table patient_address;

create table patient_address (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	address_line_1 varchar(300),
	address_line_2 varchar(300),
	city varchar(300),
	state varchar(300),
	zip_code varchar(100),
	address_type varchar(100),
	status varchar(100) not null,
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;
