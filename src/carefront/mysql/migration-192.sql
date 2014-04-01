alter table doctor add column dob_month int unsigned;
alter table doctor add column dob_year int unsigned;
alter table doctor add column dob_day int unsigned;
update doctor set dob_month=MONTH(dob), dob_year=YEAR(dob), dob_day=DAY(dob);
alter table doctor drop column dob;