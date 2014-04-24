create table drug_details (
    id int unsigned not null auto_increment,
    ndc varchar(12) not null,
    json blob not null,
    modified_date timestamp not null default CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP,
    primary key(id),
    unique key (ndc)
) character set utf8;

alter table drug_db_id modify column drug_db_id varchar(100);

