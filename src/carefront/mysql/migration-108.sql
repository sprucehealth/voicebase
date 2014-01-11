alter table patient drop column zip_code; 
create table patient_location (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	zip_code varchar(100) not null,
	city varchar(150),
	state varchar(150),
	status varchar(100) not null,
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;