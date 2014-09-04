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