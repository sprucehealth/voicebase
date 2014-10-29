create table diagnosis_intake (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null,
	question_id int unsigned not null,
	potential_answer_id int unsigned,
	answer_text mediumtext,
	layout_version_id int unsigned not null,
	answered_date timestamp not null default current_timestamp,
	status varchar(32) not null,
	doctor_id int unsigned not null,
	parent_info_intake_id int unsigned,
	parent_question_id int unsigned,
	primary key (id),
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (question_id) references question(id),
	foreign key (potential_answer_id) references potential_answer(id),
	foreign key (layout_version_id) references layout_version(id),
	foreign key (doctor_id) references doctor(id),
	foreign key (parent_info_intake_id) references diagnosis_intake(id),
	foreign key (parent_question_id) references question(id)
) character set utf8mb4;


INSERT INTO diagnosis_intake 
	(patient_visit_id, question_id, potential_answer_id, answer_text, 
		layout_version_id, answered_date, status, doctor_id, parent_info_intake_id, 
		parent_question_id)
	SELECT info_intake.context_id, info_intake.question_id, info_intake.potential_answer_id, 
	info_intake.answer_text, info_intake.layout_version_id, info_intake.answered_date, 
	info_intake.status, info_intake.role_id, info_intake.parent_info_intake_id,
	info_intake.parent_question_id
	FROM info_intake
	WHERE info_intake.role = 'DOCTOR';

DELETE FROM info_intake WHERE role = 'DOCTOR';

ALTER TABLE info_intake change column context_id patient_visit_id int unsigned not null;
ALTER TABLE info_intake add foreign key (patient_visit_id) references patient_visit(id);
ALTER TABLE info_intake change column role_id patient_id int unsigned not null;
ALTER TABLE info_intake add foreign key (patient_id) references patient(id);
ALTER TABLE info_intake drop column role;

CREATE TABLE visit_diagnosis (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null,
	status varchar(32) not null,
	diagnosis text not null,
	doctor_id int unsigned not null,
	primary key (id),
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (doctor_id) references doctor(id),
	key (doctor_id, patient_visit_id)
) character set utf8mb4;

INSERT INTO visit_diagnosis (patient_visit_id, diagnosis, status, doctor_id)
	SELECT patient_visit.id, patient_visit.diagnosis, 'ACTIVE', patient_case_care_provider_assignment.provider_id
	FROM patient_visit
	INNER JOIN patient_case_care_provider_assignment ON patient_case_care_provider_assignment.patient_case_id = patient_visit.patient_case_id
	INNER JOIN role_type on role_type.id = patient_case_care_provider_assignment.role_type_id
	WHERE patient_case_care_provider_assignment.status in ('ACTIVE', 'TEMP')
	AND role_type_tag = 'DOCTOR' 
	AND patient_visit.diagnosis is not null
	AND patient_visit.status != 'OPEN';

ALTER TABLE patient_visit drop column diagnosis;