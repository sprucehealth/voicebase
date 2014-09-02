alter table patient_receipt add column item_cost_id int unsigned;
alter table patient_receipt add foreign key (item_cost_id) references item_cost(id);
update patient_receipt set item_cost_id = (select id from item_cost where status='ACTIVE');
alter table patient_receipt modify column item_cost_id int unsigned not null;

create table doctor_transaction (
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	item_cost_id int unsigned,
	item_id int unsigned not null,
	item_type varchar(32) not null,
	patient_id int unsigned not null,
	created timestamp not null default current_timestamp, 
	foreign key (patient_id) references patient(id),
	foreign key (item_cost_id) references item_cost(id),
	foreign key (doctor_id) references doctor(id),
	primary key(id)
) character set utf8;
