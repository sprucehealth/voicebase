alter table treatment_instructions add foreign key (dr_drug_instruction_id) references dr_drug_supplemental_instruction(id);

CREATE TABLE dr_drug_supplemental_instruction_selected_state (
	id int unsigned not null auto_increment,
	drug_name_id int unsigned not null,
	drug_form_id int unsigned not null,
	drug_route_id int unsigned not null,
	doctor_id int unsigned not null,
	dr_drug_supplemental_instruction_id int unsigned not null,
	foreign key (doctor_id) references doctor(id),
	foreign key (drug_name_id) references drug_name(id),
	foreign key (drug_form_id) references drug_form(id),
	foreign key (drug_route_id) references drug_route(id),
	foreign key (dr_drug_supplemental_instruction_id) references dr_drug_supplemental_instruction(id),
	key (drug_name_id, drug_form_id, drug_route_id, doctor_id, dr_drug_supplemental_instruction_id),
	primary key (id)
) CHARACTER SET utf8;