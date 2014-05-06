create table treatment_plan_favorite_mapping (
	id int unsigned not null auto_increment,
	treatment_plan_id int unsigned not null,
	dr_favorite_treatment_plan_id int unsigned not null,
	primary key(id),
	foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade,
	foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id)
) character set utf8;

alter table treatment_instructions drop foreign key treatment_instructions_ibfk_1;
alter table treatment_instructions add foreign key (treatment_id) references treatment(id) on delete cascade;

alter table treatment_dr_template_selection drop foreign key treatment_dr_template_selection_ibfk_2;
alter table treatment_dr_template_selection add foreign key (treatment_id) references treatment(id) on delete cascade;

alter table regimen drop foreign key regimen_ibfk_3;
alter table regimen add foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade;

alter table patient_visit_follow_up drop foreign key patient_visit_follow_up_ibfk_3;
alter table patient_visit_follow_up add foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade;

alter table diagnosis_summary drop foreign key  diagnosis_summary_ibfk_3;
alter table diagnosis_summary add foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade;

alter table advice drop foreign key advice_ibfk_3;
alter table advice add foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade;

alter table treatment_drug_db_id drop foreign key treatment_drug_db_id_ibfk_1;
alter table treatment_drug_db_id add foreign key (treatment_id) references treatment(id) on delete cascade;

alter table dr_treatment_template_drug_db_id drop foreign key dr_treatment_template_drug_db_id_ibfk_1;
alter table dr_treatment_template_drug_db_id add foreign key (dr_treatment_template_id) references dr_treatment_template(id) on delete cascade;

alter table pharmacy_dispensed_treatment_drug_db_id drop foreign key pharmacy_dispensed_treatment_drug_db_id_ibfk_1;
alter table pharmacy_dispensed_treatment_drug_db_id add foreign key (pharmacy_dispensed_treatment_id) references pharmacy_dispensed_treatment(id) on delete cascade;

alter table requested_treatment_drug_db_id drop foreign key requested_treatment_drug_db_id_ibfk_1;
alter table requested_treatment_drug_db_id add foreign key (requested_treatment_id) references requested_treatment(id) on delete cascade;

alter table unlinked_dntf_treatment_drug_db_id drop foreign key unlinked_dntf_treatment_drug_db_id_ibfk_1;
alter table unlinked_dntf_treatment_drug_db_id add foreign key (unlinked_dntf_treatment_id) references unlinked_dntf_treatment(id) on delete cascade;

alter table dr_favorite_treatment_drug_db_id drop foreign key dr_favorite_treatment_drug_db_id_ibfk_1;
alter table dr_favorite_treatment_drug_db_id add foreign key (dr_favorite_treatment_id) references dr_favorite_treatment(id) on delete cascade;

alter table dr_favorite_treatment drop foreign key dr_favorite_treatment_ibfk_1;
alter table dr_favorite_treatment add foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id) on delete cascade;

alter table dr_favorite_regimen drop foreign key dr_favorite_regimen_ibfk_1;
alter table dr_favorite_regimen add foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id) on delete cascade;

alter table dr_favorite_advice drop foreign key dr_favorite_advice_ibfk_2;
alter table dr_favorite_advice add foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id) on delete cascade;

alter table dr_favorite_patient_visit_follow_up drop foreign key dr_favorite_patient_visit_follow_up_ibfk_1;
alter table dr_favorite_patient_visit_follow_up add foreign key (dr_favorite_treatment_plan_id) references dr_favorite_treatment_plan(id) on delete cascade;

alter table treatment drop foreign key treatment_ibfk_9;
alter table treatment add foreign key (treatment_plan_id) references treatment_plan(id) on delete cascade;





