alter table patient_care_provider_assignment add column health_condition_id int unsigned not null;
update patient_care_provider_assignment set health_condition_id = (select id from health_condition where health_condition_tag="health_condition_acne");
alter table patient_care_provider_assignment add foreign key (health_condition_id) references health_condition(id);