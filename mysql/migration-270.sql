alter table dr_regimen_step modify column modified_date timestamp(6) not null default current_timestamp(6) on update current_timestamp(6);
alter table dr_advice_point modify column modified_date timestamp(6) not null default current_timestamp(6) on update current_timestamp(6);
alter table unclaimed_case_queue modify column expires timestamp(6);
alter table dr_treatment_template modify column erx_last_filled_date timestamp(6) NULL;