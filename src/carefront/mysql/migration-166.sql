create table pharmacy_dispensed_treatment_drug_db_id (
	id int unsigned not null auto_increment,
	drug_db_id int unsigned not null,
	drug_db_id_tag varchar(100) not null,
	pharmacy_dispensed_treatment_id int unsigned not null,
	foreign key (pharmacy_dispensed_treatment_id) references pharmacy_dispensed_treatment(id),
	primary key(id)
 ) character set utf8;

create table unlinked_requested_treatment_drug_db_id (
	id int unsigned not null auto_increment,
	drug_db_id int unsigned not null,
	drug_db_id_tag varchar(100) not null,
	unlinked_requested_treatment_id int unsigned not null,
	foreign key (unlinked_requested_treatment_id) references unlinked_requested_treatment(id),
	primary key(id)
) character set utf8;