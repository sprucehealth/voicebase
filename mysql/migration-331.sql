CREATE TABLE account_timezone (
	account_id int unsigned not null,
	iana_timezone varchar(256) not null,
	primary key (account_id),
	foreign key (account_id) references account(id)
) character set utf8;

create table communication_snooze (
	id int unsigned not null auto_increment,
	account_id int unsigned not null,
	start_hour int unsigned not null,
	num_hours int unsigned not null,
	foreign key (account_id) references account(id),
	primary key (id)
) character set utf8;





