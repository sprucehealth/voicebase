create table cost_item (
	id int unsigned not null,
	currency varchar(10) not null,
	description varchar(300) not null,
	amount decimal(5,2) not null, 
	item_type varchar(100) not null,
	primary key (id)
) character set utf8;

create table patient_payment_receipt (
	id int unsigned not null,
	patient_id int unsigned not null,
	credit_card_id int unsigned not null,
	item_type varchar(100) not null,
	item_id int unsigned not null,
	receipt_reference_id varchar(32) not null,
	stripe_charge_id varchar(32) not null,
	creation_timestamp timestamp not null default current_timestamp,
	status varchar(32) not null,
	foreign key (patient_id) references patient(id),
	foreign key (credit_card_id) references credit_card(id),
	primary key(id)
) character set utf8;

create table patient_charge_item (
	id int unsigned not null,
	currency varchar(10) not null,
	description varchar(300) not null,
	amount decimal(5,2) not null,
	patient_payment_receipt_id int unsigned not null,
	creation_timestamp timestamp not null default current_timestamp,
	primary key (id),
	foreign key (patient_payment_receipt_id) references patient_payment_receipt(id)
) character set utf8;