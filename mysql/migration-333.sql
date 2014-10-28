alter table patient_promotion drop foreign key patient_promotion_ibfk_1;
alter table patient_promotion change column patient_id account_id int unsigned not null;
alter table patient_promotion add foreign key (account_id) references account(id);
alter table patient_promotion rename to account_promotion;

alter table patient_referral_tracking drop foreign key patient_referral_tracking_ibfk_2;
alter table patient_referral_tracking change column claiming_patient_id claiming_account_id  int unsigned not null;
alter table patient_referral_tracking add foreign key (claiming_account_id) references account(id);
alter table patient_referral_tracking rename to account_referral_tracking;

alter table patient_credit_history drop foreign key patient_credit_history_ibfk_1;
alter table patient_credit_history change column patient_id account_id int unsigned not null;
alter table patient_credit_history add foreign key (account_id) references account(id);
alter table patient_credit_history rename to account_credit_history;

alter table patient_credit drop foreign key patient_credit_ibfk_2;
alter table patient_credit change column patient_id account_id int unsigned not null;
alter table patient_credit add foreign key (account_id) references account(id);
alter table patient_credit drop foreign key patient_credit_ibfk_1;
alter table patient_credit change column last_checked_patient_credit_history_id last_checked_account_credit_history_id int unsigned not null;
alter table patient_credit add foreign key (last_checked_account_credit_history_id)  references account_credit_history(id);
alter table patient_credit rename to account_credit;

alter table parked_account change column patient_created account_created tinyint(1) not null;
