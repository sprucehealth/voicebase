alter table advice add column text varchar(150);
update advice 
	inner join dr_advice_point on dr_advice_point.id = dr_advice_point_id
	set advice.text = dr_advice_point.text;
alter table dr_advice_point add column source_id int unsigned;
alter table dr_advice_point add foreign key (source_id) references dr_advice_point(id);

alter table regimen add column text varchar(150);
update regimen
	inner join dr_regimen_step on dr_regimen_step.id = dr_regimen_step_id
	set regimen.text = dr_regimen_step.text;
alter table dr_regimen_step add column source_id int unsigned;
alter table dr_regimen_step add foreign key (source_id) references dr_regimen_step(id);


