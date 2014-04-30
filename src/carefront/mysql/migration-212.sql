create table treatment_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	treatment_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (treatment_id) references treatment(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into treatment_drug_db_ids_group_mapping (drug_db_ids_group_id, treatment_id) 
	select drug_db_ids_group_id, id from treatment where drug_db_ids_group_id is not null;
alter table treatment drop foreign key treatment_ibfk_9;
alter table treatment drop column drug_db_ids_group_id;

create table unlinked_dntf_treatment_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	unlinked_dntf_treatment_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into unlinked_dntf_treatment_drug_db_ids_group_mapping (drug_db_ids_group_id, unlinked_dntf_treatment_id) 
	select drug_db_ids_group_id, id from unlinked_dntf_treatment where drug_db_ids_group_id is not null;
alter table unlinked_dntf_treatment drop foreign key unlinked_dntf_treatment_ibfk_8;
alter table unlinked_dntf_treatment drop column drug_db_ids_group_id;

create table pharmacy_dispensed_treatment_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	pharmacy_dispensed_treatment_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (pharmacy_dispensed_treatment_id) references pharmacy_dispensed_treatment(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into pharmacy_dispensed_treatment_drug_db_ids_group_mapping (drug_db_ids_group_id, pharmacy_dispensed_treatment_id) 
	select drug_db_ids_group_id, id from pharmacy_dispensed_treatment where drug_db_ids_group_id is not null;
alter table pharmacy_dispensed_treatment drop foreign key pharmacy_dispensed_treatment_ibfk_9;
alter table pharmacy_dispensed_treatment drop column drug_db_ids_group_id;

create table requested_treatment_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	requested_treatment_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (requested_treatment_id) references requested_treatment(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into requested_treatment_drug_db_ids_group_mapping (drug_db_ids_group_id, requested_treatment_id) 
	select drug_db_ids_group_id, id from requested_treatment where drug_db_ids_group_id is not null;
alter table requested_treatment drop foreign key requested_treatment_ibfk_8;
alter table requested_treatment drop column drug_db_ids_group_id;


create table dr_treatment_template_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	dr_treatment_template_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (dr_treatment_template_id) references dr_treatment_template(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into dr_treatment_template_drug_db_ids_group_mapping (drug_db_ids_group_id, dr_treatment_template_id) 
	select drug_db_ids_group_id, id from dr_treatment_template where drug_db_ids_group_id is not null;
alter table dr_treatment_template drop foreign key dr_treatment_template_ibfk_7;
alter table dr_treatment_template drop column drug_db_ids_group_id;

create table dr_favorite_treatment_drug_db_ids_group_mapping (
	id int unsigned not null auto_increment,
	drug_db_ids_group_id int unsigned not null,
	dr_favorite_treatment_id int unsigned not null,
	foreign key (drug_db_ids_group_id) references drug_db_ids_group(id),
	foreign key (dr_favorite_treatment_id) references dr_favorite_treatment(id) on delete cascade,
	primary key(id)
) character set utf8;

insert into dr_favorite_treatment_drug_db_ids_group_mapping (drug_db_ids_group_id, dr_favorite_treatment_id) 
	select drug_db_ids_group_id, id from dr_favorite_treatment where drug_db_ids_group_id is not null;
alter table dr_favorite_treatment drop foreign key dr_favorite_treatment_ibfk_6;
alter table dr_favorite_treatment drop column drug_db_ids_group_id;







