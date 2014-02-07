alter table erx_status_events modify column id int unsigned not null auto_increment;
alter table erx_status_events modify erx_status varchar(100) not null;