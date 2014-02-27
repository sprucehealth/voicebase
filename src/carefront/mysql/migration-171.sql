drop table patient_address;

create table address (
	id int unsigned not null auto_increment,
	address_line_1 varchar(500) not null,
	address_line_2 varchar(500) not null,
	city varchar(500) not null,
	state varchar(500) not null,
	country varchar(500) not null,
	zip_code varchar(500) not null,
	primary key (id) 
) character set utf8;

create table patient_address_selection (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	address_id int unsigned not null,
	label varchar(100) not null,
	is_default tinyint(1) not null,
	is_updated_by_doctor tinyint(1) not null,
	primary key (id),
	foreign key (patient_id) references patient(id),
	foreign key (address_id) references address(id)
) character set utf8;


alter table patient add column prefix varchar(100) not null;
alter table patient add column middle_name varchar(100) not null;
alter table patient add column suffix varchar(100) not null;	