-- creating table for dispense unit ids
CREATE TABLE IF NOT EXISTS dispense_unit (
	id int unsigned NOT NULL,
	dispense_unit_text_id int unsigned not null,
	PRIMARY KEY (id),
	FOREIGN KEY (dispense_unit_text_id) REFERENCES app_text(id)
) CHARACTER SET utf8;

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Bag', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Bag'), 1, 'Bag');
insert into dispense_unit (id, dispense_unit_text_id) values (1, (select id from app_text where app_text_tag='txt_dispense_unit_Bag'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Bottle', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Bottle'), 1, 'Bottle');
insert into dispense_unit (id, dispense_unit_text_id) values (2, (select id from app_text where app_text_tag='txt_dispense_unit_Bottle'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Box', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Box'), 1, 'Box');
insert into dispense_unit (id, dispense_unit_text_id) values (3, (select id from app_text where app_text_tag='txt_dispense_unit_Box'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Capsule', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Capsule'), 1, 'Capsule');
insert into dispense_unit (id, dispense_unit_text_id) values (4, (select id from app_text where app_text_tag='txt_dispense_unit_Capsule'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Cartridge', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Cartridge'), 1, 'Cartridge');
insert into dispense_unit (id, dispense_unit_text_id) values (5, (select id from app_text where app_text_tag='txt_dispense_unit_Cartridge'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Container', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Container'), 1, 'Container');
insert into dispense_unit (id, dispense_unit_text_id) values (6, (select id from app_text where app_text_tag='txt_dispense_unit_Container'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Drop', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Drop'), 1, 'Drop');
insert into dispense_unit (id, dispense_unit_text_id) values (7, (select id from app_text where app_text_tag='txt_dispense_unit_Drop'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Gram', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Gram'), 1, 'Gram');
insert into dispense_unit (id, dispense_unit_text_id) values (8, (select id from app_text where app_text_tag='txt_dispense_unit_Gram'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Inhaler', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Inhaler'), 1, 'Inhaler');
insert into dispense_unit (id, dispense_unit_text_id) values (9, (select id from app_text where app_text_tag='txt_dispense_unit_Inhaler'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_International', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_International'), 1, 'International');
insert into dispense_unit (id, dispense_unit_text_id) values (10, (select id from app_text where app_text_tag='txt_dispense_unit_International'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Kit', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Kit'), 1, 'Kit');
insert into dispense_unit (id, dispense_unit_text_id) values (11, (select id from app_text where app_text_tag='txt_dispense_unit_Kit'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Liter', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Liter'), 1, 'Liter');
insert into dispense_unit (id, dispense_unit_text_id) values (12, (select id from app_text where app_text_tag='txt_dispense_unit_Liter'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Lozenge', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Lozenge'), 1, 'Lozenge');
insert into dispense_unit (id, dispense_unit_text_id) values (13, (select id from app_text where app_text_tag='txt_dispense_unit_Lozenge'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Milligram', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Milligram'), 1, 'Milligram');
insert into dispense_unit (id, dispense_unit_text_id) values (14, (select id from app_text where app_text_tag='txt_dispense_unit_Milligram'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Milliliter', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Milliliter'), 1, 'Milliliter');
insert into dispense_unit (id, dispense_unit_text_id) values (15, (select id from app_text where app_text_tag='txt_dispense_unit_Milliliter'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Million_Units', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Million_Units'), 1, 'Million Units');
insert into dispense_unit (id, dispense_unit_text_id) values (16, (select id from app_text where app_text_tag='txt_dispense_unit_Million_Units'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Mutually_Defined', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Mutually_Defined'), 1, 'Mutually Defined');
insert into dispense_unit (id, dispense_unit_text_id) values (17, (select id from app_text where app_text_tag='txt_dispense_unit_Mutually_Defined'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Fluid_Ounce', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Fluid_Ounce'), 1, 'Fluid Ounce');
insert into dispense_unit (id, dispense_unit_text_id) values (18, (select id from app_text where app_text_tag='txt_dispense_unit_Fluid_Ounce'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Not_Specified', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Not_Specified'), 1, 'Not Specified');
insert into dispense_unit (id, dispense_unit_text_id) values (19, (select id from app_text where app_text_tag='txt_dispense_unit_Not_Specified'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Pack', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Pack'), 1, 'Pack');
insert into dispense_unit (id, dispense_unit_text_id) values (20, (select id from app_text where app_text_tag='txt_dispense_unit_Pack'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Packet', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Packet'), 1, 'Packet');
insert into dispense_unit (id, dispense_unit_text_id) values (21, (select id from app_text where app_text_tag='txt_dispense_unit_Packet'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Pint', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Pint'), 1, 'Pint');
insert into dispense_unit (id, dispense_unit_text_id) values (22, (select id from app_text where app_text_tag='txt_dispense_unit_Pint'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Suppository', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Suppository'), 1, 'Suppository');
insert into dispense_unit (id, dispense_unit_text_id) values (23, (select id from app_text where app_text_tag='txt_dispense_unit_Suppository'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Syringe', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Syringe'), 1, 'Syringe');
insert into dispense_unit (id, dispense_unit_text_id) values (24, (select id from app_text where app_text_tag='txt_dispense_unit_Syringe'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tablespoon', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tablespoon'), 1, 'Tablespoon');
insert into dispense_unit (id, dispense_unit_text_id) values (25, (select id from app_text where app_text_tag='txt_dispense_unit_Tablespoon'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tablet', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tablet'), 1, 'Tablet');
insert into dispense_unit (id, dispense_unit_text_id) values (26, (select id from app_text where app_text_tag='txt_dispense_unit_Tablet'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Teaspoon', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Teaspoon'), 1, 'Teaspoon');
insert into dispense_unit (id, dispense_unit_text_id) values (27, (select id from app_text where app_text_tag='txt_dispense_unit_Teaspoon'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Transdermal_Patch', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Transdermal_Patch'), 1, 'Transdermal Patch');
insert into dispense_unit (id, dispense_unit_text_id) values (28, (select id from app_text where app_text_tag='txt_dispense_unit_Transdermal_Patch'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tube', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tube'), 1, 'Tube');
insert into dispense_unit (id, dispense_unit_text_id) values (29, (select id from app_text where app_text_tag='txt_dispense_unit_Tube'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Unit', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Unit'), 1, 'Unit');
insert into dispense_unit (id, dispense_unit_text_id) values (30, (select id from app_text where app_text_tag='txt_dispense_unit_Unit'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Vial', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Vial'), 1, 'Vial');
insert into dispense_unit (id, dispense_unit_text_id) values (31, (select id from app_text where app_text_tag='txt_dispense_unit_Vial'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Each', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Each'), 1, 'Each');
insert into dispense_unit (id, dispense_unit_text_id) values (32, (select id from app_text where app_text_tag='txt_dispense_unit_Each'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Gum', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Gum'), 1, 'Gum');
insert into dispense_unit (id, dispense_unit_text_id) values (33, (select id from app_text where app_text_tag='txt_dispense_unit_Gum'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Ampule', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Ampule'), 1, 'Ampule');
insert into dispense_unit (id, dispense_unit_text_id) values (34, (select id from app_text where app_text_tag='txt_dispense_unit_Ampule'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Applicator', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Applicator'), 1, 'Applicator');
insert into dispense_unit (id, dispense_unit_text_id) values (35, (select id from app_text where app_text_tag='txt_dispense_unit_Applicator'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Applicatorful', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Applicatorful'), 1, 'Applicatorful');
insert into dispense_unit (id, dispense_unit_text_id) values (36, (select id from app_text where app_text_tag='txt_dispense_unit_Applicatorful'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Bar', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Bar'), 1, 'Bar');
insert into dispense_unit (id, dispense_unit_text_id) values (37, (select id from app_text where app_text_tag='txt_dispense_unit_Bar'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Bead', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Bead'), 1, 'Bead');
insert into dispense_unit (id, dispense_unit_text_id) values (38, (select id from app_text where app_text_tag='txt_dispense_unit_Bead'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Blister', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Blister'), 1, 'Blister');
insert into dispense_unit (id, dispense_unit_text_id) values (39, (select id from app_text where app_text_tag='txt_dispense_unit_Blister'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Block', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Block'), 1, 'Block');
insert into dispense_unit (id, dispense_unit_text_id) values (40, (select id from app_text where app_text_tag='txt_dispense_unit_Block'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Bolus', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Bolus'), 1, 'Bolus');
insert into dispense_unit (id, dispense_unit_text_id) values (41, (select id from app_text where app_text_tag='txt_dispense_unit_Bolus'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Can', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Can'), 1, 'Can');
insert into dispense_unit (id, dispense_unit_text_id) values (42, (select id from app_text where app_text_tag='txt_dispense_unit_Can'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Canister', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Canister'), 1, 'Canister');
insert into dispense_unit (id, dispense_unit_text_id) values (43, (select id from app_text where app_text_tag='txt_dispense_unit_Canister'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Capler', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Capler'), 1, 'Capler');
insert into dispense_unit (id, dispense_unit_text_id) values (44, (select id from app_text where app_text_tag='txt_dispense_unit_Capler'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Carton', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Carton'), 1, 'Carton');
insert into dispense_unit (id, dispense_unit_text_id) values (45, (select id from app_text where app_text_tag='txt_dispense_unit_Carton'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Case', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Case'), 1, 'Case');
insert into dispense_unit (id, dispense_unit_text_id) values (46, (select id from app_text where app_text_tag='txt_dispense_unit_Case'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Cassette', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Cassette'), 1, 'Cassette');
insert into dispense_unit (id, dispense_unit_text_id) values (47, (select id from app_text where app_text_tag='txt_dispense_unit_Cassette'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Cylinder', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Cylinder'), 1, 'Cylinder');
insert into dispense_unit (id, dispense_unit_text_id) values (48, (select id from app_text where app_text_tag='txt_dispense_unit_Cylinder'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Disk', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Disk'), 1, 'Disk');
insert into dispense_unit (id, dispense_unit_text_id) values (49, (select id from app_text where app_text_tag='txt_dispense_unit_Disk'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Dose_Pack', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Dose_Pack'), 1, 'Dose Pack');
insert into dispense_unit (id, dispense_unit_text_id) values (50, (select id from app_text where app_text_tag='txt_dispense_unit_Dose_Pack'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Dual_Packs', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Dual_Packs'), 1, 'Dual Packs');
insert into dispense_unit (id, dispense_unit_text_id) values (51, (select id from app_text where app_text_tag='txt_dispense_unit_Dual_Packs'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Film', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Film'), 1, 'Film');
insert into dispense_unit (id, dispense_unit_text_id) values (52, (select id from app_text where app_text_tag='txt_dispense_unit_Film'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Gallon', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Gallon'), 1, 'Gallon');
insert into dispense_unit (id, dispense_unit_text_id) values (53, (select id from app_text where app_text_tag='txt_dispense_unit_Gallon'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Implant', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Implant'), 1, 'Implant');
insert into dispense_unit (id, dispense_unit_text_id) values (54, (select id from app_text where app_text_tag='txt_dispense_unit_Implant'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Inhalation', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Inhalation'), 1, 'Inhalation');
insert into dispense_unit (id, dispense_unit_text_id) values (55, (select id from app_text where app_text_tag='txt_dispense_unit_Inhalation'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Inhaler_Refill', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Inhaler_Refill'), 1, 'Inhaler Refill');
insert into dispense_unit (id, dispense_unit_text_id) values (56, (select id from app_text where app_text_tag='txt_dispense_unit_Inhaler_Refill'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Insert', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Insert'), 1, 'Insert');
insert into dispense_unit (id, dispense_unit_text_id) values (57, (select id from app_text where app_text_tag='txt_dispense_unit_Insert'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Intravenous_Bag', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Intravenous_Bag'), 1, 'Intravenous Bag');
insert into dispense_unit (id, dispense_unit_text_id) values (58, (select id from app_text where app_text_tag='txt_dispense_unit_Intravenous_Bag'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Milimeter', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Milimeter'), 1, 'Milimeter');
insert into dispense_unit (id, dispense_unit_text_id) values (59, (select id from app_text where app_text_tag='txt_dispense_unit_Milimeter'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Nebule', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Nebule'), 1, 'Nebule');
insert into dispense_unit (id, dispense_unit_text_id) values (60, (select id from app_text where app_text_tag='txt_dispense_unit_Nebule'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Needle_Free_Injection', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Needle_Free_Injection'), 1, 'Needle Free Injection');
insert into dispense_unit (id, dispense_unit_text_id) values (61, (select id from app_text where app_text_tag='txt_dispense_unit_Needle_Free_Injection'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Oscular_System', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Oscular_System'), 1, 'Oscular System');
insert into dispense_unit (id, dispense_unit_text_id) values (62, (select id from app_text where app_text_tag='txt_dispense_unit_Oscular_System'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Ounce', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Ounce'), 1, 'Ounce');
insert into dispense_unit (id, dispense_unit_text_id) values (63, (select id from app_text where app_text_tag='txt_dispense_unit_Ounce'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Pad', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Pad'), 1, 'Pad');
insert into dispense_unit (id, dispense_unit_text_id) values (64, (select id from app_text where app_text_tag='txt_dispense_unit_Pad'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Paper', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Paper'), 1, 'Paper');
insert into dispense_unit (id, dispense_unit_text_id) values (65, (select id from app_text where app_text_tag='txt_dispense_unit_Paper'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Pouch', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Pouch'), 1, 'Pouch');
insert into dispense_unit (id, dispense_unit_text_id) values (66, (select id from app_text where app_text_tag='txt_dispense_unit_Pouch'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Pound', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Pound'), 1, 'Pound');
insert into dispense_unit (id, dispense_unit_text_id) values (67, (select id from app_text where app_text_tag='txt_dispense_unit_Pound'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Puff', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Puff'), 1, 'Puff');
insert into dispense_unit (id, dispense_unit_text_id) values (68, (select id from app_text where app_text_tag='txt_dispense_unit_Puff'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Quart', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Quart'), 1, 'Quart');
insert into dispense_unit (id, dispense_unit_text_id) values (69, (select id from app_text where app_text_tag='txt_dispense_unit_Quart'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Ring', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Ring'), 1, 'Ring');
insert into dispense_unit (id, dispense_unit_text_id) values (70, (select id from app_text where app_text_tag='txt_dispense_unit_Ring'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Sachet', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Sachet'), 1, 'Sachet');
insert into dispense_unit (id, dispense_unit_text_id) values (71, (select id from app_text where app_text_tag='txt_dispense_unit_Sachet'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Scoopful', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Scoopful'), 1, 'Scoopful');
insert into dispense_unit (id, dispense_unit_text_id) values (72, (select id from app_text where app_text_tag='txt_dispense_unit_Scoopful'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Sponge', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Sponge'), 1, 'Sponge');
insert into dispense_unit (id, dispense_unit_text_id) values (73, (select id from app_text where app_text_tag='txt_dispense_unit_Sponge'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Spray', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Spray'), 1, 'Spray');
insert into dispense_unit (id, dispense_unit_text_id) values (74, (select id from app_text where app_text_tag='txt_dispense_unit_Spray'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Stick', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Stick'), 1, 'Stick');
insert into dispense_unit (id, dispense_unit_text_id) values (75, (select id from app_text where app_text_tag='txt_dispense_unit_Stick'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Strip', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Strip'), 1, 'Strip');
insert into dispense_unit (id, dispense_unit_text_id) values (76, (select id from app_text where app_text_tag='txt_dispense_unit_Strip'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Swab', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Swab'), 1, 'Swab');
insert into dispense_unit (id, dispense_unit_text_id) values (77, (select id from app_text where app_text_tag='txt_dispense_unit_Swab'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tabminder', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tabminder'), 1, 'Tabminder');
insert into dispense_unit (id, dispense_unit_text_id) values (78, (select id from app_text where app_text_tag='txt_dispense_unit_Tabminder'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tampon', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tampon'), 1, 'Tampon');
insert into dispense_unit (id, dispense_unit_text_id) values (79, (select id from app_text where app_text_tag='txt_dispense_unit_Tampon'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Tray', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Tray'), 1, 'Tray');
insert into dispense_unit (id, dispense_unit_text_id) values (80, (select id from app_text where app_text_tag='txt_dispense_unit_Tray'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Troche', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Troche'), 1, 'Troche');
insert into dispense_unit (id, dispense_unit_text_id) values (81, (select id from app_text where app_text_tag='txt_dispense_unit_Troche'));

insert into app_text (app_text_tag, comment) values ('txt_dispense_unit_Wafer', 'dispense unit');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_dispense_unit_Wafer'), 1, 'Wafer');
insert into dispense_unit (id, dispense_unit_text_id) values (82, (select id from app_text where app_text_tag='txt_dispense_unit_Wafer'));

