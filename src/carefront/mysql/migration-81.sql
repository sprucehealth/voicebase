alter table potential_answer add column status varchar(100) not null;
update potential_answer set status='ACTIVE';
