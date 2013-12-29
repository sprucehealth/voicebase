start transaction;
insert into drug_name (name) values ('Benzoyl Peroxide Topical');
insert into drug_form (name) values ('powder'),('bar'),('cream'), ('foam'),('gel'),('kit'),('liquid'),('lotion'),('pad');
insert into drug_route (name) values ('topical'), ('compounding');
commit;