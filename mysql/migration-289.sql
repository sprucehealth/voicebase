alter table treatment modify column pharmacy_notes varchar(250);
alter table unlinked_dntf_treatment modify column pharmacy_notes varchar(250);
alter table dr_treatment_template modify column pharmacy_notes varchar(250);
alter table dr_favorite_treatment modify column pharmacy_notes varchar(250);
alter table requested_treatment modify column pharmacy_notes varchar(250);
alter table pharmacy_dispensed_treatment modify column pharmacy_notes varchar(250);
