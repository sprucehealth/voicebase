alter table treatment add column is_controlled_substance tinyint;
alter table requested_treatment add column is_controlled_substance tinyint;
alter table pharmacy_dispensed_treatment add column is_controlled_substance tinyint;
alter table unlinked_dntf_treatment add column is_controlled_substance tinyint;
alter table dr_treatment_template add column is_controlled_substance tinyint;
alter table dr_favorite_treatment add column is_controlled_substance tinyint;
