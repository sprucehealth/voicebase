update layout_version set modified_date = now() where status='CREATING';

alter table patient_visit add column sku_id int unsigned;
alter table patient_visit add foreign key (sku_id) references sku(id);
update patient_visit set sku_id = (select id from sku where type='acne_visit');
alter table patient_visit modify column sku_id int unsigned not null;

insert into sku (sku_category_id, type) values ((select id from sku_category where type='visit'), 'acne_followup');

alter table layout_version add column sku_id int unsigned;
alter table layout_version add foreign key (sku_id) references sku(id);
update layout_version set sku_id = (select id from sku where type='acne_visit') where role in ('PATIENT', 'DOCTOR') and layout_purpose in ('CONDITION_INTAKE', 'REVIEW');

alter table patient_layout_version add column sku_id int unsigned;
update patient_layout_version set sku_id = (select id from sku where type='acne_visit');
alter table patient_layout_version modify column sku_id int unsigned not null;
alter table patient_layout_version add foreign key (sku_id) references sku(id);

alter table dr_layout_version add column sku_id int unsigned;
update dr_layout_version set sku_id = (select id from sku where type='acne_visit');
alter table dr_layout_version modify column sku_id int unsigned not null;
alter table dr_layout_version add foreign key (sku_id) references sku(id);

alter table app_version_layout_mapping add column sku_id int unsigned;
update app_version_layout_mapping set sku_id = (select id from sku where type='acne_visit');
alter table app_version_layout_mapping modify column sku_id int unsigned not null;
alter table app_version_layout_mapping add foreign key (sku_id) references sku(id);
alter table app_version_layout_mapping drop key layout_major;
alter table app_version_layout_mapping add unique key (layout_major, health_condition_id, platform, role, purpose, sku_id);

alter table patient_doctor_layout_mapping add column sku_id int unsigned;
update patient_doctor_layout_mapping set sku_id = (select id from sku where type='acne_visit');
alter table patient_doctor_layout_mapping modify column sku_id int unsigned not null;
alter table patient_doctor_layout_mapping add foreign key (sku_id) references sku(id);
alter table patient_doctor_layout_mapping drop key dr_major;
alter table patient_doctor_layout_mapping add unique key (dr_major, dr_minor, patient_major, patient_minor, health_condition_id, sku_id);





















