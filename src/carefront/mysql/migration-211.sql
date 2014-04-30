alter table pharmacy_dispensed_treatment_drug_db_id modify column drug_db_id varchar(100) not null;
alter table requested_treatment_drug_db_id modify column drug_db_id_tag varchar(100) not null;
alter table unlinked_dntf_treatment_drug_db_id modify column drug_db_id varchar(100) not null;
alter table unlinked_dntf_treatment_drug_db_id modify column drug_db_id_tag varchar(100) not null;