alter table treatment modify column days_supply int unsigned;
alter table pharmacy_dispensed_treatment modify column days_supply int unsigned;
alter table requested_treatment modify column days_supply int unsigned;
alter table unlinked_dntf_treatment modify column days_supply int unsigned;