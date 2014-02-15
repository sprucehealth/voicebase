create table dr_demo_whitelist {
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	foreign key (doctor_id) references doctor(id),
	primary key (id)
	}