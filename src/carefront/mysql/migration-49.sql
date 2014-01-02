CREATE TABLE drug_name (
	id int unsigned not null auto_increment,
	name varchar(150) not null,
	primary key (id)
) CHARACTER SET utf8;

CREATE TABLE drug_form (
	id int unsigned not null auto_increment,
	name varchar(150) not null,
	primary key (id)
) CHARACTER SET utf8;

CREATE TABLE drug_route (
	id int unsigned not null auto_increment,
	name varchar(150) not null,
	primary key (id)
) CHARACTER SET utf8;

create table drug_supplemental_instruction (
	id int unsigned not null auto_increment,
	text varchar(150) not null,
	drug_name_id int unsigned not null,
	drug_form_id int unsigned,
	drug_route_id int unsigned,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (drug_name_id) references drug_name(id),
	foreign key (drug_form_id) references drug_form(id),
	foreign key (drug_route_id) references drug_route(id),
	primary key (id)
) CHARACTER set utf8;

create table dr_drug_supplemental_instruction (
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	text varchar(150) not null,
	drug_name_id int unsigned not null,
	drug_form_id int unsigned,
	drug_route_id int unsigned,
	selected tinyint(1) not null,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (drug_form_id) references drug_form(id),
	foreign key (drug_route_id) references drug_route(id),
	foreign key (drug_name_id) references drug_name(id),
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) CHARACTER set utf8;