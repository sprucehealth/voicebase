create table doctor_queue (
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	status varchar(100) not null,
	event_type varchar(100) not null,
	enqueue_date timestamp not null default current_timestamp,
	completed_date timestamp,
	item_id int unsigned not null,
	primary key (id),
	foreign key (doctor_id) references doctor(id)
) CHARACTER set utf8;