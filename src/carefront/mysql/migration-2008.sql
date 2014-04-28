create table drug_db_ids_group (
	id int unsigned not null auto_increment,
	creation_date timestamp(6) not null default current_timestamp(6),
	treatment_id int unsigned not null,
	foreign key (treatment_id) references treatment(id),
	primary key(id)
) character set utf8;


-- Step 1: Create a mapping between drug_db_ids_group and treatment
insert into drug_db_ids_group (treatment_id) select distinct treatment_id from drug_db_id; 
alter table treatment add column drug_db_ids_group_id int unsigned;
update treatment 
	inner join drug_db_ids_group on drug_db_ids_group.treatment_id = treatment.id
	set drug_db_ids_group_id = drug_db_ids_group.id;		
alter table treatment modify column drug_db_ids_group_id int unsigned not null;
alter table treatment add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);

-- Step 2: Create a mapping between drug_db_ids and drug_db_ids_group
alter table drug_db_id add column drug_db_ids_group_id int unsigned;
update drug_db_id 
	inner join drug_db_ids_group on drug_db_ids_group.treatment_id = drug_db_id.treatment_id
	set drug_db_ids_group_id = drug_db_ids_group.id;	
alter table drug_db_id modify column drug_db_ids_group_id int unsigned not null;
alter table drug_db_id add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);

-- Step 3: Remove mapping between drug_db_ids_group and treatment given that link lives in the treatment table
alter table drug_db_ids_group drop foreign key drug_db_ids_group_ibfk_1;
alter table drug_db_ids_group drop column treatment_id;	

-- Step 4: Remove treatment_id from drug_db_id given that the link is between the drug_db_ids_group and treatment
alter table drug_db_id drop foreign key drug_db_id_ibfk_1;
alter table drug_db_id drop column treatment_id;


-- Move over the drug_db_ids for UNLINKED_DNTF_TREATMENT

-- Step 1: Create temporary link in drug_db_ids_group to unlinked_dntf_treatment
alter table drug_db_ids_group add column unlinked_dntf_treatment_id int unsigned;
alter table drug_db_ids_group add foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id);

-- Step 2: Create group ids for the number of groups coming in from unlinked dntf treatment
insert into drug_db_ids_group (unlinked_dntf_treatment_id) select distinct unlinked_dntf_treatment_id from unlinked_dntf_treatment_drug_db_id;
alter table unlinked_dntf_treatment add column drug_db_ids_group_id int unsigned;
update unlinked_dntf_treatment 
	inner join drug_db_ids_group on drug_db_ids_group.unlinked_dntf_treatment_id = unlinked_dntf_treatment.id
	set drug_db_ids_group_id = drug_db_ids_group.id;
alter table unlinked_dntf_treatment add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);


-- Step 3: Insert into drug_db_ids from unlinked_dntf_treatment_drug_db_ids;
insert into drug_db_id (drug_db_id_tag, drug_db_id, drug_db_ids_group_id) 
	select unlinked_dntf_treatment_drug_db_id.drug_db_id_tag, unlinked_dntf_treatment_drug_db_id.drug_db_id, unlinked_dntf_treatment.drug_db_ids_group_id from unlinked_dntf_treatment_drug_db_id
		inner join unlinked_dntf_treatment on unlinked_dntf_treatment.id = unlinked_dntf_treatment_drug_db_id.unlinked_dntf_treatment_id
		inner join drug_db_ids_group on unlinked_dntf_treatment.drug_db_ids_group_id = drug_db_ids_group.id;

-- Step 4: Remove unlinked_dntf_treatment_drug_db_id;
drop table unlinked_dntf_treatment_drug_db_id;
alter table drug_db_ids_group drop foreign key drug_db_ids_group_ibfk_1;
alter table drug_db_ids_group drop column unlinked_dntf_treatment_id;


-- Move over the drug_db_ids for PHARMACY_DISPENSED_TREATMENT

-- Step 1: Create temporary link in drug_db_ids_group to unlinked_dntf_treatment
alter table drug_db_ids_group add column pharmacy_dispensed_treatment_id int unsigned;
alter table drug_db_ids_group add foreign key (pharmacy_dispensed_treatment_id) references pharmacy_dispensed_treatment(id);

-- Step 2: Create group ids for the number of groups coming in from unlinked dntf treatment
insert into drug_db_ids_group (pharmacy_dispensed_treatment_id) select distinct pharmacy_dispensed_treatment_id from pharmacy_dispensed_treatment_drug_db_id;
alter table pharmacy_dispensed_treatment add column drug_db_ids_group_id int unsigned;
update pharmacy_dispensed_treatment 
	inner join drug_db_ids_group on drug_db_ids_group.pharmacy_dispensed_treatment_id = pharmacy_dispensed_treatment.id
	set drug_db_ids_group_id = drug_db_ids_group.id;
alter table pharmacy_dispensed_treatment modify column drug_db_ids_group_id int unsigned not null;
alter table pharmacy_dispensed_treatment add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);


-- Step 3: Insert into drug_db_ids from unlinked_dntf_treatment_drug_db_ids;
insert into drug_db_id (drug_db_id_tag, drug_db_id, drug_db_ids_group_id) 
	select pharmacy_dispensed_treatment_drug_db_id.drug_db_id_tag, pharmacy_dispensed_treatment_drug_db_id.drug_db_id, pharmacy_dispensed_treatment.drug_db_ids_group_id from pharmacy_dispensed_treatment_drug_db_id
		inner join pharmacy_dispensed_treatment on pharmacy_dispensed_treatment.id = pharmacy_dispensed_treatment_drug_db_id.pharmacy_dispensed_treatment_id
		inner join drug_db_ids_group on pharmacy_dispensed_treatment.drug_db_ids_group_id = drug_db_ids_group.id;

-- Step 4: Remove unlinked_dntf_treatment_drug_db_id;
drop table pharmacy_dispensed_treatment_drug_db_id;
alter table drug_db_ids_group drop foreign key drug_db_ids_group_ibfk_1;
alter table drug_db_ids_group drop column pharmacy_dispensed_treatment_id;

-- Move over the drug_db_ids for REQUESTED_TREATMENT

-- Step 1: Create temporary link in drug_db_ids_group to unlinked_dntf_treatment
alter table drug_db_ids_group add column requested_treatment_id int unsigned;
alter table drug_db_ids_group add foreign key (requested_treatment_id) references requested_treatment(id);

-- Step 2: Create group ids for the number of groups coming in from unlinked dntf treatment
insert into drug_db_ids_group (requested_treatment_id) select distinct requested_treatment_id from requested_treatment_drug_db_id;
alter table requested_treatment add column drug_db_ids_group_id int unsigned;
update requested_treatment 
	inner join drug_db_ids_group on drug_db_ids_group.requested_treatment_id = requested_treatment.id
	set drug_db_ids_group_id = drug_db_ids_group.id;
alter table requested_treatment modify column drug_db_ids_group_id int unsigned not null;
alter table requested_treatment add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);

-- Step 3: Insert into drug_db_ids from unlinked_dntf_treatment_drug_db_ids;
insert into drug_db_id (drug_db_id_tag, drug_db_id, drug_db_ids_group_id) 
	select requested_treatment_drug_db_id.drug_db_id_tag, requested_treatment_drug_db_id.drug_db_id, requested_treatment.drug_db_ids_group_id from requested_treatment_drug_db_id
		inner join requested_treatment on requested_treatment.id = requested_treatment_drug_db_id.requested_treatment_id
		inner join drug_db_ids_group on requested_treatment.drug_db_ids_group_id = drug_db_ids_group.id;

-- Step 4: Remove unlinked_dntf_treatment_drug_db_id;
drop table requested_treatment_drug_db_id;
alter table drug_db_ids_group drop foreign key drug_db_ids_group_ibfk_1;
alter table drug_db_ids_group drop column requested_treatment_id;