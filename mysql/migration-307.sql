alter table layout_version add column major int unsigned not null;
alter table layout_version add column minor int unsigned not null;
alter table layout_version add column patch int unsigned not null;

alter table patient_layout_version drop foreign key patient_layout_version_ibfk_5;
alter table patient_layout_version drop column object_storage_id;

alter table patient_layout_version add column major int unsigned not null;
alter table patient_layout_version add column minor int unsigned not null;
alter table patient_layout_version add column patch int unsigned not null;

create table diagnosis_layout_version (
	id int unsigned not null auto_increment,
	layout_version_id int unsigned not null,
	layout_blob_storage_id int unsigned not null,
	health_condition_id int unsigned not null,
	major int unsigned not null,
	minor int unsigned not null,
	patch int unsigned not null,
	language_id int unsigned not null,
	status varchar(64) not null,
	modified timestamp not null default current_timestamp on update current_timestamp,
	created timestamp not null default current_timestamp,
	foreign key (layout_version_id) references layout_version(id),
	foreign key (layout_blob_storage_id) references layout_blob_storage(id),
	foreign key (health_condition_id) references health_condition(id),
	foreign key (language_id) references languages_supported(id),
	primary key(id)
) character set utf8;

-- Move all doctor layout versions pertaining to the diagnosis into this newly created table
insert into diagnosis_layout_version (layout_version_id, layout_blob_storage_id, health_condition_id, status, modified, created, major, minor, patch, language_id) 
	select layout_version_id, layout_blob_storage_id, health_condition_id, status, modified_date, creation_date, 0,0,0,1 from dr_layout_version 
		where layout_version_id in (select id from layout_version where role='DOCTOR' and layout_purpose = 'DIAGNOSE' and layout_blob_storage_id is not null);

-- Delete all layout versions from dr_layout_version pertaining to diagnosis
delete from dr_layout_version where layout_version_id in (select id from layout_version where layout_purpose='DIAGNOSE' and role='DOCTOR');

alter table dr_layout_version drop foreign key dr_layout_version_ibfk_4;
alter table dr_layout_version drop column object_storage_id;

alter table dr_layout_version add column major int unsigned not null;
alter table dr_layout_version add column minor int unsigned not null;
alter table dr_layout_version add column patch int unsigned not null;

alter table dr_layout_version add column language_id int unsigned;
update dr_layout_version set language_id = 1;
alter table dr_layout_version modify column language_id int unsigned not null;
alter table dr_layout_version add foreign key (language_id) references languages_supported(id);

create table patient_doctor_layout_mapping (
	id int unsigned not null auto_increment,
	dr_major int unsigned not null,
	dr_minor int unsigned not null,
	patient_major int unsigned not null,
	patient_minor int unsigned not null,
	health_condition_id int unsigned not null,
	created timestamp not null default current_timestamp,
	foreign key (health_condition_id) references health_condition(id),
	unique key (dr_major, dr_minor, patient_major, patient_minor),
	primary key (id)
) character set utf8;

create table app_version_layout_mapping (
	id int unsigned not null auto_increment,
	app_major int unsigned not null,
	app_minor int unsigned not null,
	app_patch int unsigned not null,
	layout_major int unsigned not null,
	health_condition_id int unsigned not null,
	platform varchar(64) not null,
	role varchar(64) not null,
	purpose varchar(64) not null,	
	unique key (layout_major, health_condition_id, platform, role, purpose),
	foreign key (health_condition_id) references health_condition(id),	
	primary key(id)
) character set utf8;