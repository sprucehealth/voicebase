CREATE TABLE diagnosis_code (
	id int unsigned not null auto_increment,
	code varchar(32) not null,
	name varchar(1024) not null,
	billable tinyint(1) not null,
	unique key (code),
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE diagnosis_includes_note (
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE diagnosis_inclusion_term (
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE diagnosis_excludes1_note (
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE diagnosis_excludes2_note (
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE diagnosis_use_additional_code_note (
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE diagnosis_code_first_note(
	id int unsigned not null auto_increment,
	diagnosis_code_id int unsigned not null,
	note varchar(1024) not null,
	foreign key (diagnosis_code_id) references diagnosis_code(id) on delete cascade,
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE diagnosis_details_layout_template (
	id int unsigned not null auto_increment,
	type varchar(64) not null,
	layout blob not null,
	diagnosis_code_id int unsigned not null,
	major int unsigned not null,
	minor int unsigned not null,
	patch int unsigned not null,
	active tinyint(1) not null,
	created timestamp not null default current_timestamp,
	unique key (diagnosis_code_id, major, minor, patch),
	key (diagnosis_code_id, active),
	foreign key (diagnosis_code_id) references diagnosis_code(id),
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE diagnosis_details_layout (
	id int unsigned not null auto_increment,
	type varchar(64) not null,
	layout blob not null,
	diagnosis_code_id int unsigned not null,
	major int unsigned not null,
	minor int unsigned not null,
	patch int unsigned not null,
	active tinyint(1) not null,
	created timestamp not null default current_timestamp,
	template_layout_id int unsigned not null,
	unique key (diagnosis_code_id, major, minor, patch),
	foreign key (diagnosis_code_id) references diagnosis_code(id),
	foreign key (template_layout_id) references diagnosis_details_layout_template(id),
	primary key (id)
)  ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 	

CREATE TABLE visit_diagnosis_set (
	id int unsigned not null auto_increment,
	patient_visit_id int unsigned not null,
	doctor_id int unsigned not null,
	notes text not null,
	unsuitable tinyint(1) not null,
	unsuitable_reason text not null,
	active tinyint(1) not null default 0,
	created timestamp not null default current_timestamp,
	foreign key (patient_visit_id) references patient_visit(id),
	foreign key (doctor_id) references doctor(id),
	key (patient_visit_id, active),
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 

CREATE TABLE visit_diagnosis_item (
	id int unsigned not null auto_increment,
	visit_diagnosis_set_id int unsigned not null,
	diagnosis_code_id int unsigned not null,
	layout_version_id int unsigned,
	foreign key (visit_diagnosis_set_id) references visit_diagnosis_set(id),
	foreign key (diagnosis_code_id) references diagnosis_code(id),
	foreign key (layout_version_id) references diagnosis_details_layout(id),
	primary key (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;  

CREATE TABLE diagnosis_details_intake (
	id int unsigned not null auto_increment,
	doctor_id int unsigned not null,
	visit_diagnosis_item_id int unsigned not null,
	layout_version_id int unsigned not null,
	answered_date timestamp not null default current_timestamp,
	question_id int unsigned not null,
	potential_answer_id int unsigned ,
	answer_text text,
	parent_question_id int unsigned,
	parent_info_intake_id int unsigned,
	client_clock varchar(128),
	foreign key (doctor_id) references doctor(id),
	foreign key (visit_diagnosis_item_id) references visit_diagnosis_item(id),
	foreign key (layout_version_id) references diagnosis_details_intake(id),
	foreign key (question_id) references question(id),
	foreign key (potential_answer_id) references potential_answer(id),
	foreign key (parent_question_id) references question(id),
	foreign key (parent_info_intake_id) references diagnosis_details_intake(id),
	primary key(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4; 	
