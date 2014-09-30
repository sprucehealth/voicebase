alter table person drop key role_type;
alter table person add unique key (role_type_id, role_id);