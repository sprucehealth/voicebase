alter table treatment add column drug_name_id int unsigned;
alter table treatment add column drug_form_id int unsigned;
alter table treatment add column drug_route_id int unsigned;
alter table treatment add foreign key (drug_name_id) references drug_name(id);
alter table treatment add foreign key (drug_form_id) references drug_form(id);
alter table treatment add foreign key (drug_route_id) references drug_route(id);