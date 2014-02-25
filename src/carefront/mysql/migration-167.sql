create table deny_refill_reason (
	id int unsigned not null auto_increment,
	reason_code varchar(100) not null,
	reason varchar(150)  not null,
	primary key (id)
) character set utf8;

insert into deny_refill_reason (reason_code, reason) values ('AA', 'Patient unknown to the provider'), 
	('AB', 'Patient never under provider care'),
	('AC', 'Patient no longer under provider care'),
	('AD', 'Refill too soon'),
	('AE', 'Medication never prescribed for patient'),
	('AF', 'Patient should contact provider'),
	('AG', 'Refill not appropriate'),
	('AH', 'Patient has picked up prescription'),
	('AJ', 'Patient has picked up partial fill of prescription'),
	('AK', 'Patient has not picked up prescription, drug returned to stock'),
	('AL', 'Change not appropriate'),
	('AM', 'Patient needs appointment'),
	('AN', 'Prescriber not associated with this practice or location'),
	('AO', 'No attempt will be made to obtain Prior Authorization'),
	('AP', 'Request already responded to by other means (e.g. phone or fax)');