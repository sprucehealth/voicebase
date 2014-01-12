create table patient_agreement (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	agreement_type varchar(100) not null,
	status varchar(100) not null,
	agreement_date timestamp not null default current_timestamp,
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;