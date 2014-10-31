create table account_app_version (
	account_id int unsigned not null,
	major int unsigned not null,
	minor int unsigned not null,
	patch int unsigned not null,
	platform varchar(32) not null,
	platform_version varchar(32) not null,
	device varchar(128) not null,
	device_model varchar(128) not null,
	last_modified_date timestamp default current_timestamp on update current_timestamp,
	primary key (account_id, platform, device, device_model),
	foreign key (account_id) references account(id)
) character set utf8;