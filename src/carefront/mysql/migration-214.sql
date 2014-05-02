alter table dr_regimen_step add index (doctor_id);
alter table dr_advice_point add index (doctor_id);
alter table treatment_plan_favorite_mapping add unique key (treatment_plan_id, dr_favorite_treatment_plan_id);