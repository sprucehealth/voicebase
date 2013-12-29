start transaction;

insert into drug_supplemental_instruction (text, drug_name_id, status) values ('Benzoyl peroxide level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), 'ACTIVE');
insert into drug_supplemental_instruction (text, drug_name_id, status) values ('Benzoyl peroxide level instruction 2', (select id from drug_name where name='Benzoyl Peroxide Topical'), 'ACTIVE');

insert into drug_supplemental_instruction (text, drug_name_id, drug_route_id, status) values ('Benzoyl peroxide and route topical level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), (select id from drug_route where name='topical'), 'ACTIVE');
insert into drug_supplemental_instruction (text, drug_name_id, drug_route_id, status) values ('Benzoyl peroxide and route compounding level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), (select id from drug_route where name='compounding'), 'ACTIVE');

insert into drug_supplemental_instruction (text, drug_name_id, drug_route_id, drug_form_id, status) values ('Benzoyl peroxide, route topical and form cream level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), (select id from drug_route where name='topical'), (select id from drug_form where name='cream'), 'ACTIVE');
insert into drug_supplemental_instruction (text, drug_name_id, drug_route_id, drug_form_id, status) values ('Benzoyl peroxide, route topical and form gel level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), (select id from drug_route where name='topical'), (select id from drug_form where name='gel'), 'ACTIVE');
insert into drug_supplemental_instruction (text, drug_name_id, drug_route_id, drug_form_id, status) values ('Benzoyl peroxide, route topical and form liquid level instruction 1', (select id from drug_name where name='Benzoyl Peroxide Topical'), (select id from drug_route where name='topical'), (select id from drug_form where name='liquid'), 'ACTIVE');



commit;