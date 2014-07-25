alter table regimen modify column text varchar(1024) not null;
alter table advice modify column text varchar(1024) not null;
alter table info_intake modify column answer_text varchar(1024);