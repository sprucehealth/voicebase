create table migrations (
	migration_id int unsigned not null,
	migration_date timestamp not null default current_timestamp,
	migration_user varchar(100) not null
) character set utf8;