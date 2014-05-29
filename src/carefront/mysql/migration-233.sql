create table notification_prompt_status (
	id int unsigned not null auto_increment,
	account_id int unsigned not null,	
	prompt_status varchar(100) not null,
	creation_date timestamp not null default current_timestamp,
	primary key(id),
	foreign key (account_id) references account(id)
) character set utf8;

insert into notification_prompt_status (account_id, prompt_status) 
	select account_id, prompt_status from patient_prompt_status
		inner join patient on patient.id = patient_id;

drop table patient_prompt_status;
