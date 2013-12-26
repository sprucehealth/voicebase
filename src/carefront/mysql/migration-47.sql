alter table drug_db_id drop foreign key drug_db_id_ibfk_1;
alter table treatment drop foreign key treatment_ibfk_1;
alter table treatment modify column id int unsigned not null auto_increment;
alter table treatment_plan  modify column id int unsigned not null auto_increment;
alter table drug_db_id modify column id int unsigned not null auto_increment; 
alter table treatment add foreign key (treatment_plan_id) references treatment_plan(id);
alter table drug_db_id add foreign key (treatment_id) references treatment(id);