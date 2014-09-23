alter table care_provider_state_elligibility add column notify tinyint(1) not null default 0;
alter table care_provider_state_elligibility add unique key (provider_id, role_type_id, care_providing_state_id);
