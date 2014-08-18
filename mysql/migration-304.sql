create table item_cost (
	id int unsigned not null auto_increment,
	item_type varchar(100) not null,
	status varchar(32) not null,
	primary key (id)
) character set utf8;

create table cost_line_item (
	id int unsigned not null auto_increment,
	currency varchar(10) not null,
	description varchar(300) not null,
	amount decimal(5,2) not null, 
	item_cost_id int unsigned not null,
	foreign key (item_cost_id) references item_cost(id),
	primary key (id)
) character set utf8;

create table patient_receipt (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	credit_card_id int unsigned,
	item_type varchar(100) not null,
	item_id int unsigned not null,
	receipt_reference_id varchar(32) not null,
	stripe_charge_id varchar(32),
	creation_timestamp timestamp not null default current_timestamp,
	last_modified_timestamp timestamp not null default current_timestamp on update current_timestamp,
	status varchar(32) not null,
	unique key (patient_id, item_id, item_type),
	foreign key (patient_id) references patient(id),
	foreign key (credit_card_id) references credit_card(id),
	primary key(id)
) character set utf8;

create table patient_charge_item (
	id int unsigned not null auto_increment,
	currency varchar(10) not null,
	description varchar(300) not null,
	amount decimal(5,2) not null,
	patient_receipt_id int unsigned not null,
	creation_timestamp timestamp not null default current_timestamp,
	primary key (id),
	foreign key (patient_receipt_id) references patient_receipt(id)
) character set utf8;