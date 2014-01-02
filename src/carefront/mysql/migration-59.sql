
create table advice_point (
	id int unsigned not null auto_increment,
	text varchar(150) not null,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	primary key (id)
) CHARACTER set utf8;


create table dr_advice_point (
	id int unsigned not null auto_increment,
	text varchar(150) not null,
	doctor_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) CHARACTER set utf8;


create table advice (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null,
	dr_advice_point_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (dr_advice_point_id) references dr_advice_point(id),
	primary key (id)
) CHARACTER set utf8;

create table patient_visit_follow_up (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null,
	doctor_id int unsigned not null,
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) CHARACTER Set utf8;