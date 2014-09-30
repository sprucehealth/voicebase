create table doctor_case_notification (
	doctor_id int unsigned not null,
	last_notified timestamp not null default current_timestamp,
	key (last_notified),
	primary key (doctor_id)
) character set utf8;

create table care_providing_state_notification (
	care_providing_state_id int unsigned not null,
	last_notified timestamp not null default current_timestamp,
	primary key (care_providing_state_id),
	key (last_notified)
) character set utf8;