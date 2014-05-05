create table dr_favorite_treatment_plan (
	id int unsigned not null auto_increment,
	name varchar(600) not null,
	doctor_id int unsigned not null,
	modified_date timestamp(6) not null default current_timestamp(6) on update current_timestamp(6),
	foreign key (doctor_id) references doctor(id),
	primary key (id)
) character set utf8;

create table dr_favorite_treatment (
	id int unsigned not null auto_increment,
	dr_favorite_treatment_plan_id int unsigned not null,
	drug_internal_name varchar(250) not null,
	dispense_value decimal(21,10) not null,
	dispense_unit_id int unsigned not null,
	refills int unsigned not null,
	substitutions_allowed tinyint,
	days_supply int unsigned,
	pharmacy_notes varchar(150),
	patient_instructions varchar(150) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	status varchar(100) not null,
	dosage_strength varchar(250) not null,
	type varchar(150) not null,
	drug_name_id int unsigned,
	drug_form_id int unsigned,
	drug_route_id int unsigned,
	primary key(id),
	foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id),
	foreign key (dispense_unit_id) references dispense_unit(id),
	foreign key (drug_name_id) references drug_name(id),
	foreign key (drug_route_id) references drug_route(id),
	foreign key (drug_form_id) references drug_form(id)
) character set utf8;

create table dr_favorite_treatment_drug_db_id (
	id int unsigned not null auto_increment,
	drug_db_id varchar(100) not null,
	drug_db_id_tag varchar(100) not null,
	dr_favorite_treatment_id int unsigned not null,
	foreign key (dr_favorite_treatment_id) references dr_favorite_treatment(id),
	primary key(id)	
) character set utf8;

create table dr_favorite_regimen (
	id int unsigned not null auto_increment,
	regimen_type varchar(150) not null,
	status varchar(100) not null, 
	creation_date timestamp(6) not null default current_timestamp(6),
	text varchar(150) not null,
	dr_regimen_step_id int unsigned not null,
	dr_favorite_treatment_plan_id int unsigned not null,
	primary key(id),
	foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id),
	foreign key (dr_regimen_step_id) references dr_regimen_step(id)
) character set utf8;

create table dr_favorite_advice (
	id int unsigned not null auto_increment,
	status varchar(100) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	text varchar(150) not null,
	dr_advice_point_id int unsigned not null,
	dr_favorite_treatment_plan_id int unsigned not null,
	primary key(id),
	foreign key (dr_advice_point_id) references dr_advice_point(id),
	foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id)
) character set utf8;

create table dr_favorite_patient_visit_follow_up (
	id int unsigned not null auto_increment,
	follow_up_date date not null,
	follow_up_value int unsigned not null,
	follow_up_unit varchar(100) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	status varchar(100) not null,
	dr_favorite_treatment_plan_id int unsigned not null,
	foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id),
	primary key(id)
) character set utf8;

