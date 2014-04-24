alter table patient_pharmacy_selection modify column pharmacy_id varchar(300) NOT NULL;
alter table patient_address_selection modify column label varchar(100) NOT NULL;
alter table rx_refill_request modify column comments varchar(500) NOT NULL;
alter table rx_refill_status_events modify column event_details varchar(500) NOT NULL;
alter table patient modify column prefix varchar(100) NOT NULL;
alter table patient modify column suffix varchar(100) NOT NULL;
alter table patient modify column middle_name varchar(100) NOT NULL;