rename table provider_role to role_type;
alter table role_type change column provider_tag role_type_tag varchar(250) not null;
alter table care_provider_state_elligibility change column provider_role_id role_type_id int unsigned not null;
alter table patient_care_provider_assignment change column provider_role_id role_type_id int unsigned not null;
alter table patient_visit_care_provider_assignment change column provider_role_id role_type_id int unsigned not null;

insert into role_type (role_type_tag) values ('PATIENT');

alter table account add unique key (email);
alter table account add column role_type_id int unsigned not null;
update account set role_type_id = (select id from role_type where role_type_tag='PATIENT') where id in (select account_id from patient);
update account set role_type_id = (select id from role_type where role_type_tag='DOCTOR') where id in (select account_id from doctor);
alter table account add foreign key (role_type_id) references role_type(id);

alter table account add registration_date timestamp(6) not null default current_timestamp(6);
alter table account add column last_opened_date timestamp(6) not null default current_timestamp(6);

alter table person add column role_type_id int unsigned not null;
update person
	inner join role_type on role_type_tag = person.role_type 
 		set role_type_id = role_type.id;
alter table person add foreign key (role_type_id) references role_type(id);
alter table person drop column role_type;	

alter table auth_token add unique key (account_id);
alter table auth_token modify column created timestamp(6) not null default current_timestamp(6);