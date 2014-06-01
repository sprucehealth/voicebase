create table photo_slot_type (
	id int unsigned not null auto_increment,
	slot_type varchar(100) not null,
	primary key (id)
) character set utf8;

create table photo_slot (
	id int unsigned not null auto_increment,
	question_id int unsigned not null,
	slot_name_app_text_id int unsigned not null,
	slot_type_id int unsigned not null,
	required tinyint(1) not null,
	status varchar(100) not null,
	primary key (id),
	foreign key (question_id) references question(id),
	foreign key (slot_name_app_text_id) references app_text(id),
	foreign key (slot_type_id) references photo_slot_type(id)
) character set utf8;

create table photo_intake_section (
	id int unsigned not null auto_increment,
	section_name varchar(150) not null, 
	question_id int unsigned not null,
	patient_id int unsigned not null,
	patient_visit_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	primary key (id),
	foreign key (question_id) references question(id),
	foreign key (patient_id) references patient(id),
	foreign key (patient_visit_id) references patient_visit(id)
) character set utf8;

create table photo_intake_slot (
	id int unsigned not null auto_increment,
	photo_slot_id int unsigned not null, 
	photo_id int unsigned not null,
	photo_slot_name varchar(150) not null,
	photo_intake_section_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	primary key(id),
	foreign key (photo_slot_id) references photo_slot(id),
	foreign key (photo_id) references photo(id),
	foreign key (photo_intake_section_id) references photo_intake_section(id) on delete cascade
) character set utf8;