alter table layout_version add layout_purpose varchar(250);

update layout_version set layout_purpose='CONDITION_INTAKE' where role='PATIENT';
update layout_version set layout_purpose='REVIEW' where role='DOCTOR';
