create table if not exists question_fields (
	id int unsigned not null auto_increment,
	question_field varchar(250) not null,
	question_id int unsigned not null,
	app_text_id int unsigned not null,
	foreign key (question_id) references question(id),
	foreign key (app_text_id) references app_text(id),
	primary key (id)
) character set utf8;
