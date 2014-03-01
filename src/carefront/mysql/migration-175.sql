create table pending_task (
	id int unsigned not null auto_increment,
	type varchar(100) not null,
	item_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	primary key(id)
) character set utf8;