alter table drug_db_id add column treatment_id int unsigned not null;
alter table drug_db_id add foreign key (treatment_id) references treatment(id);