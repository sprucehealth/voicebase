CREATE TABLE treatment_instructions (
	id int unsigned not null auto_increment,
	treatment_id int unsigned not null,
	dr_drug_instruction_id int unsigned not null,
	status varchar(100) not null,
	foreign key (treatment_id) references treatment(id),
	primary key (id)
) CHARACTER SET utf8;