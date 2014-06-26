create table extra_question_fields (
	id int unsigned not null auto_increment,
	question_id int unsigned not null,
	json blob not null,
	foreign key (question_id) references question(id),	
	primary key (id),
	unique key (question_id)
) character set utf8;

insert into extra_question_fields (question_id, json) values (
	(select id from question where question_tag='q_other_location_photo_section'),
	'{"allows_multiple_sections":true, "user_defined_section_title":true}');
