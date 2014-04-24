alter table state change full_name abbreviation_tmp varchar(10) not null;
alter table state change abbreviation full_name varchar(300) not null;
alter table state change abbreviation_tmp abbreviation varchar(10) not null;
