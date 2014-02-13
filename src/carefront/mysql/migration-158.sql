create table pharmacy_selection(
	id int unsigned not null auto_increment,
	pharmacy_id varchar(500),
	address_line_1 varchar(500),
	address_line_2 varchar(500),
	source varchar(100) not null,
	city varchar(100),
	state varchar(100),
	country varchar(100),
	phone varchar(100),
	zip_code varchar(100),
	lat varchar(100),
	lng varchar(100),
	name varchar(500),
	primary key(id)
 ) character set utf8;

alter table treatment add column pharmacy_id int unsigned;
alter table treatment add foreign key (pharmacy_id) references pharmacy_selection(id);

insert into pharmacy_selection (pharmacy_id, address_line_1, source, city, state, country, phone, zip_code, lat, lng, name) 
	select distinct pharmacy_id, address, source, city, state, country, phone, zip_code, lat, lng, name from patient_pharmacy_selection;

alter table patient_pharmacy_selection add column pharmacy_selection_id int unsigned;
alter table patient_pharmacy_selection add foreign key (pharmacy_selection_id) references pharmacy_selection(id);
update patient_pharmacy_selection 
	inner join pharmacy_selection on pharmacy_selection.pharmacy_id = patient_pharmacy_selection.pharmacy_id set pharmacy_selection_id = pharmacy_selection.id ;


alter table patient_pharmacy_selection drop column source;
alter table patient_pharmacy_selection drop column city;
alter table patient_pharmacy_selection drop column state;
alter table patient_pharmacy_selection drop column country;
alter table patient_pharmacy_selection drop column address;
alter table patient_pharmacy_selection drop column phone;
alter table patient_pharmacy_selection drop column zip_code;
alter table patient_pharmacy_selection drop column lat;
alter table patient_pharmacy_selection drop column lng;
alter table patient_pharmacy_selection drop column name;	