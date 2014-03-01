create table credit_card (
	id int unsigned not null auto_increment,
	third_party_card_id varchar(100),
	type varchar(100) not null,
	patient_id int unsigned not null,
	address_id int unsigned not null,
	is_default tinyint(1) not null,
	label varchar(200),
	status varchar(100) not null,
	foreign key (address_id) references address(id),	 	
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;