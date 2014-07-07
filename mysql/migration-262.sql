create table case_notification (
	id int unsigned not null auto_increment,
	patient_case_id int unsigned not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	notification_type varchar(100) not null,
	item_id int unsigned not null,
	data blob not null,
	foreign key (patient_case_id) references patient_case(id),
	primary key (id)
) character set utf8;