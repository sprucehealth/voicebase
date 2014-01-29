alter table treatment modify column treatment_plan_id int unsigned;
create table dr_favorite_treatment (
	id int unsigned not null auto_increment,
	name varchar(600) not null,
	doctor_id int unsigned not null,
	treatment_id int unsigned not null,
	status varchar(100) not null,
	foreign key (doctor_id) references doctor(id),
	foreign key (treatment_id) references treatment(id),
	primary key (id)
) character set utf8;