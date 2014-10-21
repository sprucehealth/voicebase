create table sku_category (
	id int unsigned not null auto_increment,
	type varchar(32) not null,
	unique key (type),
	primary key (id)
) character set utf8;

create table sku (
	id int unsigned not null auto_increment,
	sku_category_id int unsigned not null,
	type varchar(32) not null,
	unique key (type),
	foreign key (sku_category_id) references sku_category(id),
	primary key (id)
) character set utf8;


insert into sku_category (type) values ('visit');
insert into sku (sku_category_id, type) values (1, 'acne_visit');

alter table item_cost drop column item_type;
alter table item_cost add column sku_id int unsigned;
update item_cost set sku_id = (select id from sku where type = 'acne_visit');
alter table item_cost add foreign key (sku_id) references sku(id);
alter table item_cost modify column sku_id int unsigned not null;

alter table patient_receipt drop column item_type;
alter table patient_receipt add column sku_id int unsigned; 
update patient_receipt set sku_id = (select id from sku where type = 'acne_visit');
alter table patient_receipt modify column sku_id int unsigned not null;
alter table patient_receipt add foreign key (sku_id) references sku(id);
alter table patient_receipt add unique key (patient_id, item_id, sku_id);

alter table doctor_transaction drop column item_type;
alter table doctor_transaction add column sku_id int unsigned not null;
update doctor_transaction set sku_id = (select id from sku where type = 'acne_visit');
alter table doctor_transaction add foreign key (sku_id) references sku(id);
alter table doctor_transaction modify column sku_id int unsigned not null;

create table promo_code_prefix (
	prefix varchar(32) not null,
	status varchar(32) not null,
	created timestamp not null default current_timestamp,
	primary key (prefix)
) character set utf8;


create table promotion_group (
	id int unsigned not null auto_increment,
	name varchar(32) not null,
	max_allowed_promos int not null,
	unique key (name),
	primary key(id)
) character set utf8;


create table promotion_code (
	id int unsigned not null auto_increment,
	code varchar(32) not null,
	is_referral tinyint(1) not null,
	unique key (code),
	primary key (id)
) character set utf8;

create table promotion (
	promotion_code_id int unsigned not null,
	promo_type varchar(32) not null,
	promo_data blob not null,
	promotion_group_id int unsigned not null,
	expires timestamp default null,
	created timestamp not null default current_timestamp,
	foreign key (promotion_group_id) references promotion_group(id),
	foreign key (promotion_code_id) references promotion_code(id),
	primary key (promotion_code_id)
) character set utf8;


create table parked_account (
	id int unsigned not null auto_increment,
	email varchar(250) not null,
	state varchar(250) not null,
	promotion_code_id int unsigned not null,
	patient_created tinyint(1) not null,
	created timestamp not null default current_timestamp,
	last_modified_time timestamp not null default current_timestamp on update current_timestamp,
	unique key (email),
	foreign key (promotion_code_id) references promotion_code(id),
	primary key (id)
) character set utf8;


create table patient_promotion (
	patient_id int unsigned not null,
	promotion_code_id int unsigned not null,
	promotion_group_id int unsigned not null,
	promo_type varchar(32) not null,
	promo_data blob not null,
	expires timestamp default null,
	created timestamp not null default current_timestamp,
	status varchar(32) not null,
	foreign key (patient_id) references patient(id),
	foreign key (promotion_code_id) references promotion_code(id), 
	foreign key (promotion_group_id) references promotion_group(id),
	primary key (patient_id, promotion_code_id)
) character set utf8;


create table referral_program_template (
	id int unsigned not null auto_increment,
	referral_type varchar(32) not null,
	referral_data blob not null,
	created timestamp not null default current_timestamp,
	status varchar(32) not null,
	role_type_id int unsigned not null,
	foreign key (role_type_id) references role_type(id),
	key (role_type_id, status),
	primary key (id)
) character set utf8;

create table referral_program (
	referral_program_template_id int unsigned,
	account_id int unsigned not null,
	promotion_code_id int unsigned not null,
	referral_type varchar(32) not null,
	referral_data blob not null,
	created timestamp not null default current_timestamp,
	status varchar(32) not null,
	foreign key (account_id) references account(id),
	foreign key (promotion_code_id) references promotion_code(id),
	foreign key (referral_program_template_id) references referral_program_template(id),
	primary key (account_id, promotion_code_id)
) character set utf8;

create table patient_referral_tracking (
	promotion_code_id int unsigned not null,
	claiming_patient_id int unsigned not null,
	referring_account_id int unsigned not null,
	created timestamp not null default current_timestamp,
	status varchar(32) not null,
	foreign key (referring_account_id) references account(id),
	foreign key (claiming_patient_id) references patient(id),
	foreign key (promotion_code_id) references promotion_code(id),
	primary key (claiming_patient_id)
) character set utf8;

create table patient_credit_history (
	id int unsigned not null auto_increment,
	patient_id int unsigned not null,
	credit int not null,
	description varchar(256) not null,
	created timestamp not null default current_timestamp,
	foreign key (patient_id) references patient(id),
	primary key(id)
) character set utf8;

create table patient_credit (
	patient_id int unsigned not null,
	credit int unsigned not null,
	last_checked_patient_credit_history_id int unsigned not null,
	last_modified_time timestamp not null default current_timestamp on update current_timestamp,
	foreign key (last_checked_patient_credit_history_id) references patient_credit_history(id),
	foreign key (patient_id) references patient(id),
	primary key (patient_id)
) character set utf8;