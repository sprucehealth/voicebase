alter table patient add column erx_patient_id int unsigned;
alter table treatment add column erx_sent_date timestamp;
alter table treatment add column erx_id int unsigned;
alter table treatment_plan add column doctor_id int unsigned;
alter table treatment_plan add foreign key (doctor_id) references doctor(id);

create table erx_status_events (
	id int unsigned not null,
	treatment_id int unsigned not null,
	erx_status int unsigned not null,
	creation_date timestamp not null default current_timestamp,
	foreign key (treatment_id) references treatment(id),
	primary key (id)
) character set utf8;