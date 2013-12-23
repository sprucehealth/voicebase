-- creating table for doctor diagnosis intae
CREATE TABLE IF NOT EXISTS dr_diagnosis_intake (
	id int unsigned NOT NULL AUTO_INCREMENT,
	patient_visit_id int unsigned not null,
	doctor_id int unsigned not null,
	question_id int unsigned not null,
	potential_answer_id int unsigned,
	answer_text varchar(600),
	layout_version_id int unsigned not null,
	answered_date timestamp not null default current_timestamp,
	status varchar(100) not null,
	parent_diagnosis_intake int unsigned,
	parent_question_id int unsigned,
	foreign key (parent_diagnosis_intake) references dr_diagnosis_intake(id),
	foreign key (parent_question_id) references question(id),
	foreign key (layout_version_id) references layout_version(id),
	foreign key (potential_answer_id) references potential_answer(id),
	foreign key (question_id) references question(id),
	foreign key (doctor_id) references doctor(id),
	foreign key (patient_visit_id) references patient_visit(id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;