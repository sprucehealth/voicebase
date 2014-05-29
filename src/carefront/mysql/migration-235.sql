create table layout_blob_storage (
	id int unsigned not null auto_increment,
	layout blob not null,
	creation_date timestamp not null default current_timestamp,
	primary key(id)
) character set utf8;

alter table layout_version drop foreign key layout_version_ibfk_2;
alter table layout_version modify column object_storage_id int unsigned;
alter table layout_version add foreign key (object_storage_id) references object_storage(id);
alter table layout_version add column layout_blob_storage_id int unsigned;
alter table layout_version add foreign key (layout_blob_storage_id) references layout_blob_storage(id); 
