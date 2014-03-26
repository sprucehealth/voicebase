
create table unlinked_dntf_treatment (
	id int unsigned not null auto_increment,
	drug_internal_name varchar(250) not null,
	dispense_value int unsigned not null,
	dispense_unit_id int unsigned not null,
	refills int unsigned not null,
	days_supply int unsigned,
	pharmacy_notes varchar(150) not null,
	patient_instructions varchar(150) not null,
	creation_date timestamp(6) default current_timestamp(6),
	status varchar(100) not null,
	dosage_strength varchar(250) not null,
	type varchar(150) not null,
	drug_name_id int unsigned,
	drug_form_id int unsigned,
	drug_route_id int unsigned,
	erx_sent_date timestamp,
	erx_id int unsigned,
	pharmacy_id int unsigned,
	erx_last_filled_date timestam
	patient_id int unsigned not null,
	doctor_id int unsigned not null,
	primary key(id),
	foreign key (dispense_unit_id) references dispense_unit(id),
	foreign key (drug_name_id) references drug_name(id),
	foreign key (drug_form_id) references drug_form(id),
	foreign key (drug_route_id) references drug_route(id),
	foreign key (pharmacy_id) references pharmacy_selection(id),
	foreign key (patient_id) references patient(id),
	foreign key (doctor_id) references doctor(id)
) character set utf8;

create table unlinked_dntf_treatment_status_events (
	id int unsigned not null,
	unlinked_dntf_treatment_id int unsigned not null,
	erx_status varchar(100) not null,
	creation_date timestamp(6) default current_timestamp(6),
	status varchar(100) not null,
	event_details varchar(500),
	reported_timestamp timestamp(6),
	primary key(id),
	foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id)
) character set utf8;

create table unlinked_dntf_treatment_drug_db_id (
	id int unsigned not null auto_increment,
	drug_db_id int unsigned not null,
	drug_db_id_tag int unsigned not null,
	unlinked_dntf_treatment_id int unsigned not null,
	primary key (id),
	foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id)
) character set utf8;

create table dntf_mapping (
	id int unsigned not null auto_increment,
	treatment_id int unsigned,
	unlinked_dntf_treatment_id int unsigned,
	rx_refill_request_id int unsigned not null,
	foreign key (treatment_id) references treatment(id),
	foreign key (rx_refill_request_id) references rx_refill_request(id),
	foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id),
	primary key(id)
) character set utf8;

