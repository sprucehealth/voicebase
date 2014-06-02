CREATE TABLE resource_guide_section (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	ordinal INT NOT NULL,
	title VARCHAR(256) NOT NULL,
	PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE resource_guide (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	section_id INT UNSIGNED NOT NULL,
	ordinal INT NOT NULL,
	title VARCHAR(256) NOT NULL,
	photo_url VARCHAR(256) NOT NULL,
	layout BLOB NOT NULL,
	creation_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	modified_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	FOREIGN KEY (section_id) REFERENCES resource_guide_section (id),
	PRIMARY KEY (id)
) CHARACTER SET utf8;

<<<<<<< HEAD
=======
create table photo_intake_section (
	id int unsigned not null auto_increment,
	section_name varchar(150) not null, 
	question_id int unsigned not null,
	patient_id int unsigned not null,
	patient_visit_id int unsigned not null,
	status varchar(100) not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	primary key (id),
	foreign key (question_id) references question(id),
	foreign key (patient_id) references patient(id),
	foreign key (patient_visit_id) references patient_visit(id)
) character set utf8;

create table photo_intake_slot (
	id int unsigned not null auto_increment,
	photo_slot_id int unsigned not null, 
	photo_id int unsigned not null,
	photo_slot_name varchar(150) not null,
	photo_intake_section_id int unsigned not null,
	creation_date timestamp(6) not null default current_timestamp(6),
	primary key(id),
	foreign key (photo_slot_id) references photo_slot(id),
	foreign key (photo_id) references photo(id),
	foreign key (photo_intake_section_id) references photo_intake_section(id) on delete cascade
) character set utf8;
>>>>>>> - creating a handler for photo intake
