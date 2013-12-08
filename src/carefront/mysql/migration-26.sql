CREATE TABLE IF NOT EXISTS doctor (
	id int unsigned NOT NULL AUTO_INCREMENT,
	first_name varchar(250) not null,
	last_name varchar(250) not null,
	gender varchar(250) not null,
	dob date not null,
	account_id int unsigned not null,
	dea_number varchar(250),
	npi_number varchar(250),
	foreign key (account_id) references account(id),
	primary key (id)
) character set utf8;

create table if not exists provider_role (
	id int unsigned not null AUTO_INCREMENT,
	provider_tag varchar(250) not null,
	primary key(id)
) character set utf8;

create table if not exists care_providing_state (
	id int unsigned not null AUTO_INCREMENT,
	state varchar(100) not null,
	health_condition_id int unsigned not null,
	primary key (id),
	foreign key (health_condition_id) references health_condition(id)
) character set utf8;

create table if not exists care_provider_state_elligibility (
	id int unsigned not null AUTO_INCREMENT,
	provider_role_id int unsigned not null,
	provider_id int unsigned not null,
	care_providing_state_id int unsigned not null,
	primary key (id),
	foreign key (provider_role_id) references provider_role(id),
	foreign key (care_providing_state_id) references care_providing_state(id)
) character set utf8;

create table if not exists patient_visit_provider_group (
	id int unsigned not null AUTO_INCREMENT,
	patient_visit_id int unsigned not null,
	created_date timestamp not null default current_timestamp,
	modified_date timestamp not null on update current_timestamp,
	status varchar(250) not null,
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (id)
) character set utf8;

create table if not exists patient_visit_provider_assignment (
	id int unsigned not null AUTO_INCREMENT,
	patient_visit_id int unsigned not null,
	provider_role_id int unsigned not null,
	provider_id int unsigned not null,
	assignment_group_id int unsigned not null,
	status varchar(250) not null,
	primary key (id),
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (assignment_group_id) references patient_visit_provider_group(id),
	foreign key (provider_role_id) references provider_role(id)
) character set utf8;

create table if not exists diagnosis_type (
	id int unsigned not null AUTO_INCREMENT,
	tag varchar(250) not null,
	diagnosis_text_id int unsigned not null,
	health_condition_id int unsigned not null,
	foreign key (diagnosis_text_id) references app_text(id),
	foreign key (health_condition_id) references health_condition(id),
	primary key (id)
) character set utf8;

create table if not exists diagnosis_strength (
	id int unsigned not null AUTO_INCREMENT,
	tag varchar (250) not null,
	diagnosis_type_id int unsigned not null,
	strength_title_text_id int unsigned not null,
	strength_description_text_id int unsigned not null,
	primary key(id),
	foreign key (diagnosis_type_id) references diagnosis_type(id),
	foreign key (strength_title_text_id) references app_text(id),
	foreign key (strength_description_text_id) references app_text(id)
) character set utf8;

create table if not exists patient_diagnosis (
	id int unsigned not null AUTO_INCREMENT,
	diagnosis_type_id int unsigned not null,
	diagnosis_strength_id int unsigned not null,
	patient_visit_id int unsigned not null,
	diagnosis_date timestamp not null default current_timestamp,
	status varchar(250) not null,
	foreign key (diagnosis_type_id) references diagnosis_type(id),
	foreign key (diagnosis_strength_id) references diagnosis_strength(id),
	foreign key (patient_visit_id) references patient_visit(id),
	primary key (id)
) character set utf8;


