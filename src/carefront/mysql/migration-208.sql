alter table dr_treatment_template add column drug_internal_name varchar(250);
alter table dr_treatment_template add column dispense_value decimal(21,10);
alter table dr_treatment_template add column dispense_unit_id int unsigned;
alter table dr_treatment_template add column refills int unsigned;
alter table dr_treatment_template add column substitutions_allowed tinyint;
alter table dr_treatment_template add column days_supply int unsigned;
alter table dr_treatment_template add column pharmacy_notes varchar(150);
alter table dr_treatment_template add column patient_instructions varchar(150);
alter table dr_treatment_template add column dosage_strength varchar(250);
alter table dr_treatment_template add column type varchar(150);
alter table dr_treatment_template add column drug_name_id int unsigned;
alter table dr_treatment_template add column drug_form_id int unsigned;
alter table dr_treatment_template add column drug_route_id int unsigned;
alter table dr_treatment_template add column erx_sent_date timestamp;
alter table dr_treatment_template add column erx_id int unsigned;
alter table dr_treatment_template add column pharmacy_id int unsigned;
alter table dr_treatment_template add column erx_last_filled_date timestamp;
alter table dr_treatment_template add column drug_db_ids_group_id int unsigned;
alter table dr_treatment_template add column creation_date timestamp(6) not null default current_timestamp(6);

update dr_treatment_template inner join treatment on dr_treatment_template.treatment_id = treatment.id
	set dr_treatment_template.drug_internal_name = treatment.drug_internal_name,
	dr_treatment_template.dispense_value = treatment.dispense_value,
	dr_treatment_template.dispense_unit_id = treatment.dispense_unit_id,
	dr_treatment_template.refills = treatment.refills,
	dr_treatment_template.substitutions_allowed = treatment.substitutions_allowed,
	dr_treatment_template.days_supply = treatment.days_supply,
	dr_treatment_template.pharmacy_notes = treatment.pharmacy_notes,
	dr_treatment_template.patient_instructions = treatment.patient_instructions,
	dr_treatment_template.dosage_strength = treatment.dosage_strength,
	dr_treatment_template.type = treatment.type,
	dr_treatment_template.drug_name_id = treatment.drug_name_id,
	dr_treatment_template.drug_form_id = treatment.drug_form_id,
	dr_treatment_template.drug_route_id = treatment.drug_route_id,
	dr_treatment_template.erx_sent_date = treatment.erx_sent_date,
	dr_treatment_template.erx_id = treatment.erx_id,
	dr_treatment_template.pharmacy_id = treatment.pharmacy_id,
	dr_treatment_template.erx_last_filled_date = treatment.erx_last_filled_date,
	dr_treatment_template.drug_db_ids_group_id = treatment.drug_db_ids_group_id
		where treatment.treatment_plan_id is null;	
	
alter table dr_treatment_template drop foreign key dr_treatment_template_ibfk_2;

delete t.* from treatment t
	inner join dr_treatment_template on dr_treatment_template.treatment_id = t.id 
	and t.treatment_plan_id is null;

alter table treatment drop foreign key treatment_ibfk_4;
alter table treatment modify column treatment_plan_id int unsigned not null;
alter table treatment add foreign key (treatment_plan_id) references treatment_plan(id);

alter table dr_treatment_template drop column treatment_id;

-- clean up dr_treatment_templates that are no longer used
delete from treatment_dr_template_selection 
	where dr_treatment_template_id in (select id from dr_treatment_template where status='DELETED');
delete from dr_treatment_template where status='DELETED';

alter table dr_treatment_template modify column drug_internal_name varchar(250) not null;
alter table dr_treatment_template modify column dispense_value decimal(21,10) not null;
alter table dr_treatment_template modify column dispense_unit_id int unsigned not null;
alter table dr_treatment_template modify column refills int unsigned not null;
alter table dr_treatment_template modify column substitutions_allowed tinyint not null;
alter table dr_treatment_template modify column patient_instructions varchar(150) not null;
alter table dr_treatment_template modify column dosage_strength varchar(250) not null;
alter table dr_treatment_template modify column type varchar(150) not null;
alter table dr_treatment_template modify column drug_name_id int unsigned not null;

alter table dr_treatment_template add foreign key (dispense_unit_id) references dispense_unit(id);
alter table dr_treatment_template add foreign key (drug_name_id) references drug_name(id);
alter table dr_treatment_template add foreign key (drug_route_id) references drug_route(id);
alter table dr_treatment_template add foreign key (drug_form_id) references drug_form(id);
alter table dr_treatment_template add foreign key (pharmacy_id) references pharmacy_selection(id);
alter table dr_treatment_template add foreign key (drug_db_ids_group_id) references drug_db_ids_group(id);




