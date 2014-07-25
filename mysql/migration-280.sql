alter table regimen modify column text varchar(2048) not null;
alter table regimen_step modify column text varchar(2048) not null;
alter table dr_favorite_regimen modify column text varchar(2048) not null;
alter table dr_regimen_step modify column text varchar(2048) not null;

alter table advice modify column text varchar(2048) not null;
alter table advice_point modify column text varchar(2048) not null;
alter table dr_advice_point modify column text varchar(2048) not null;
alter table dr_favorite_advice modify column text varchar(2048) not null;

alter table info_intake modify column answer_text varchar(2048);