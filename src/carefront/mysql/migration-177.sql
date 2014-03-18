alter table credit_card modify column creation_date timestamp(6) default current_timestamp(6);
alter table doctor_queue modify column enqueue_date timestamp(6) default current_timestamp(6);
alter table advice modify column creation_date timestamp(6) default current_timestamp(6);
alter table advice_point modify column creation_date timestamp(6) default current_timestamp(6);
alter table dr_advice_point modify column creation_date timestamp(6) default current_timestamp(6);
alter table dr_drug_supplemental_instruction modify column creation_date timestamp(6) default current_timestamp(6);
alter table dr_layout_version modify column creation_date timestamp(6) default current_timestamp(6);
alter table dr_regimen_step modify column creation_date timestamp(6) default current_timestamp(6);
alter table drug_supplemental_instruction modify column creation_date timestamp(6) default current_timestamp(6);
alter table erx_status_events modify column creation_date timestamp(6) default current_timestamp(6);
alter table info_intake modify column answered_date timestamp(6) default current_timestamp(6);
alter table layout_version modify column creation_date timestamp(6) default current_timestamp(6);
alter table migrations modify column migration_date timestamp(6) default current_timestamp(6);
alter table object_storage modify column creation_date timestamp(6) default current_timestamp(6);
alter table patient_agreement modify column agreement_date timestamp(6) default current_timestamp(6);
alter table patient_care_provider_group modify column created_date timestamp(6) default current_timestamp(6);
alter table patient_diagnosis modify column diagnosis_date timestamp(6) default current_timestamp(6);
alter table patient_layout_version modify column creation_date timestamp(6) default current_timestamp(6);
alter table patient_visit modify column creation_date timestamp(6) default current_timestamp(6);
alter table patient_visit_care_provider_assignment modify column assignment_date timestamp(6) default current_timestamp(6);
alter table pending_task modify column creation_date timestamp(6) default current_timestamp(6);
alter table pharmacy_dispensed_treatment modify column creation_date timestamp(6) default current_timestamp(6);
alter table regimen modify column creation_date timestamp(6) default current_timestamp(6);
alter table regimen_step modify column creation_date timestamp(6) default current_timestamp(6);
alter table treatment modify column creation_date timestamp(6) default current_timestamp(6);
alter table treatment_plan modify column creation_date timestamp(6) default current_timestamp(6);








