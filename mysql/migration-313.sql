alter table patient add column training tinyint(1) not null;

create table training_case_set (
	id int unsigned not null auto_increment,
	created timestamp not null default current_timestamp,
	status varchar(32) not null,
	primary key(id)
) character set utf8;
 

create table training_case (
	training_case_set_id int unsigned not null,
	patient_visit_id int unsigned not null,
	template_name varchar(64) not null,
	created timestamp not null default current_timestamp,
	foreign key (training_case_set_id) references training_case_set(id) on delete cascade,
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (patient_visit_id)
) character set utf8;
