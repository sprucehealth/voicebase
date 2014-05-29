alter table dr_layout_version drop foreign key dr_layout_version_ibfk_2;
alter table dr_layout_version modify column object_storage_id int unsigned;
alter table dr_layout_version add foreign key (object_storage_id) references object_storage(id);

alter table dr_layout_version add column layout_blob_storage_id int unsigned;
alter table dr_layout_version add foreign key (layout_blob_storage_id) references layout_blob_storage(id); 


alter table patient_layout_version drop foreign key patient_layout_version_ibfk_3;
alter table patient_layout_version modify column object_storage_id int unsigned;
alter table patient_layout_version add foreign key (object_storage_id) references object_storage(id);

alter table patient_layout_version add column layout_blob_storage_id int unsigned;
alter table patient_layout_version add foreign key (layout_blob_storage_id) references layout_blob_storage(id); 