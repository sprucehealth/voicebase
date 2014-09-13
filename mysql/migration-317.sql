create table scheduled_message (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	message_type varchar(64) not null,
	message_json blob not null,
	event varchar(64) not null,
	status varchar(64) not null,
	created timestamp not null default current_timestamp,
	scheduled timestamp default current_timestamp,
	available_after timestamp,
	completed timestamp,
	error varchar(512),
	foreign key (patient_id) references patient(id),
	primary key (id)
) character set utf8;

create table scheduled_message_template (
	id int unsigned not null auto_increment,
	name text not null,
	event varchar(64) not null,
	schedule_period int unsigned not null,
	message text not null,
	creator_account_id int unsigned not null,
	created timestamp not null default current_timestamp,
	foreign key (creator_account_id) references account(id),
	primary key (id)
) character set utf8;