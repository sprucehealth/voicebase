alter table case_notification change item_id uid varchar(100) not null;
alter table case_notification add unique key (uid);