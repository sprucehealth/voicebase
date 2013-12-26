-- creating table to treatments

CREATE TABLE IF NOT EXISTS drug_db_id	(
	id int unsigned NOT NULL,
	drug_db_id_tag varchar(100) NOT NULL,
	drug_db_id int unsigned NOT NULL,
	treatment_id int unsigned not null,
	PRIMARY KEY (id),
	foreign key (treatment_id) references treatment(id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS treatment_plan (
	id int unsigned not null,
	status varchar(100) not null,
	patient_visit_id int unsigned not null,
	primary key(id),
	foreign key (patient_visit_id) references patient_visit(id)
) CHARACTER SET utf8;

CREATE TABLE IF NOT EXISTS treatment (
	id int unsigned not null,
	treatment_plan_id int unsigned not null,
	additional_drug_db_ids int unsigned not null,
	drug_internal_name varchar(250) not null,
	dispense_value int unsigned not null,
	dispense_unit_id int unsigned not null,
	refills int unsigned not null,
	substitutions_allowed tinyint(1) default NULL,
	days_supply int unsigned not null,
	pharmacy_notes varchar(150) not null,
	patient_instructions varchar(150) not null,
	creation_date timestamp not null default current_timestamp,
	status varchar(100) not null,
	primary key(id),
	foreign key (treatment_plan_id) references treatment_plan(id),
	foreign key (additional_drug_db_ids) references drug_db_id(id),
	foreign key (dispense_unit_id) references dispense_unit(id)
) CHARACTER SET utf8;
 	