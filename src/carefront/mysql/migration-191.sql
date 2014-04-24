alter table patient add column dob_month int unsigned;
alter table patient add column dob_year int unsigned;
alter table patient add column dob_day int unsigned;
update patient set dob_month=MONTH(dob), dob_day=DAY(dob), dob_year=YEAR(dob);
alter table patient drop column dob;
