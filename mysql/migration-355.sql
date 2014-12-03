-- Get rid of the status column now that we are only storing the latest answer for each question
-- and also update foreign key so that when an info_intake entry is deleted, so is any other
-- info_intake entry that has its parent as the info_intake entry being deleted
alter table info_intake drop foreign key info_intake_ibfk_7;
alter table info_intake add foreign key (parent_info_intake_id) references info_intake(id) on delete cascade;
delete from info_intake where status='INACTIVE';
alter table info_intake drop column status;

-- Similarly, get rid of status column and update foreign key
alter table diagnosis_intake drop foreign key diagnosis_intake_ibfk_6;
alter table diagnosis_intake add foreign key (parent_info_intake_id) references info_intake(id) on delete cascade;
delete from diagnosis_intake where status='INACTIVE';
alter table diagnosis_intake drop column status;

-- Get rid of status column and delete photo slot entries when a section is deleted
delete from photo_intake_slot where photo_intake_section_id in (select distinct id from photo_intake_section where status='INACTIVE');
alter table photo_intake_slot drop foreign key  photo_intake_slot_ibfk_3;
alter table photo_intake_slot add foreign key (photo_intake_section_id) references photo_intake_section(id) on delete cascade;
delete from photo_intake_section where status='INACTIVE';
alter table photo_intake_section drop column status;

-- Add the client clock to the existing intake tables
alter table info_intake add column client_clock varchar(128);
alter table diagnosis_intake add column client_clock varchar(128);
alter table photo_intake_section add column client_clock varchar(128);