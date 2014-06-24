create table unclaimed_case_queue (
  id int unsigned not null auto_increment,
  care_providing_state_id int unsigned not null, 
  event_type varchar(100) not null,
  item_id int unsigned not null,
  patient_case_id int unsigned not null,
  status varchar(100) not null,
  locked tinyint(1) not null default 0,
  doctor_id int unsigned,
  enqueue_date timestamp(6) not null default current_timestamp(6),
  expires timestamp(6) not null default current_timestamp(6),
  foreign key (care_providing_state_id) references care_providing_state(id),
  foreign key (patient_case_id) references patient_case(id),
  foreign key (doctor_id) references doctor(id),
  primary key (id),
  key (care_providing_state_id),
  unique key (patient_case_id)
 ) character set utf8;

 alter table patient_case_care_provider_assignment add column expires timestamp(6);
 alter table patient_care_provider_assignment add column expires timestamp(6);