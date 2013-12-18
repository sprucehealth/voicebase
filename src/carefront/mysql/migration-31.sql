start transaction;
insert into provider_role (provider_tag) values ('PRIMARY_DOCTOR');
commit;

alter table doctor add provider_role_id int unsigned;
alter table doctor add foreign key (provider_role_id) references provider_role(id);