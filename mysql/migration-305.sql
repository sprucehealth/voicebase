create table account_config (
	account_id int unsigned not null,
	platform varchar(128) not null,
	extended_auth tinyint(1) not null,
	primary key (account_id, platform),
	foreign key (account_id) references account(id)
) character set utf8;	

