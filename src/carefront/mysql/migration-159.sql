create table treatment_dr_favorite_selection (
	id int unsigned not null auto_increment,
	treatment_id int unsigned not null,
	dr_favorite_treatment_id int unsigned not null,
	foreign key (dr_favorite_treatment_id) references dr_favorite_treatment(id),
	foreign key (treatment_id) references treatment(id),
	primary key (id)
) character set utf8;	