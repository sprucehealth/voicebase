alter table dr_regimen_step add column modified_date timestamp(6) on update current_timestamp(6);
alter table dr_advice_point add column modified_date timestamp(6) on update current_timestamp(6);