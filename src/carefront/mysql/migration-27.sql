alter table care_providing_state add long_state varchar(250) not null;

start transaction;

insert into care_providing_state (state, long_state, health_condition_id) values ("CA", "California", (select id from health_condition where health_condition_tag = "health_condition_acne"));
insert into provider_role (provider_tag) values ("DOCTOR");
commit;