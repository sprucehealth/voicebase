alter table doctor add column short_display_name varchar(300);
update doctor set short_display_name = CONCAT("Dr. ", last_name);
alter table doctor modify column short_display_name varchar(300) not null;

alter table doctor add column long_display_name varchar(600) not null;
update doctor set long_display_name = CONCAT("Dr. ", first_name, " ", last_name);
alter table doctor modify column long_display_name varchar(300) not null;
