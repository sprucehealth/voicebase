create table pharmacy (
	id serial primary key,
	ncpdpid integer not null,
	store_number text not null,
	store_name text not null,
	address_line_1 text not null,
	address_line_2 text,
	city text not null,
	state text not null,
	zip text not null,
	phone_primary text not null,
	fax text not null,
	active_start_time timestamp not null,
	active_end_time timestamp not null,
	service_level integer,
	last_modified_date timestamp not null,
	twenty_four_hour_flag text,
	version text not null,
	is_from_surescripts boolean not null
);
\copy pharmacy FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/pharmacy.csv' WITH DELIMITER  ','  CSV HEADER;


-- Import results from smarty streets
create table pharmacy_smartystreets_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	zip_precision text
);

\copy pharmacy_smartystreets_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-first-50000-results-short.txt';
\copy pharmacy_smartystreets_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-remaining-results-short.txt';

-- Add ncpdpid so as to correctly identify pharmacies
alter table pharmacy_smartystreets_location add column ncpdpid integer;

update pharmacy_smartystreets_location
 	set ncpdpid = pharmacy.ncpdpid
 	from pharmacy where pharmacy.id = pharmacy_smartystreets_location.id;

-- Create table into which all unique smarty streets addresses would go
create table pharmacy_ss_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	zip_precision text,
	ncpdpid integer
);

-- Insert location information for unique pharmaciesd only 
insert into pharmacy_ss_location (id, latitude, longitude, zip_precision, ncpdpid) select distinct on (ncpdpid) id, latitude, longitude, zip_precision, ncpdpid from pharmacy_smartystreets_location order by ncpdpid;

-- Drop the temporarly helper table
drop table pharmacy_smartystreets_location;


-- Import results from maplarge
create table pharmacy_maplarge_location (
	id integer references pharmacy(id),
	latitude float8,
	longitude float8,
	matchtype text,
	numtype text,
	ncpdpid integer
);

-- Create temporary table in which to insert results with extraneous data
create table pharmacy_maplarge_temp_data (
	latitude numeric(10,6),
	longitude numeric(10,6),
	matchtype text,
	numtype text,
	id integer references pharmacy(id),
	address_line_1 text,
	city text,
	state text,
	zip text
);

\copy pharmacy_maplarge_temp_data FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/maplarge_results_unique_ncpdpid.csv' WITH DELIMITER  ','  CSV HEADER QUOTE '"';

-- Copy over results into table to contain only relevant fields
insert into pharmacy_maplarge_location (id, latitude, longitude, matchtype, numtype) select id, latitude, longitude, matchtype, numtype from pharmacy_maplarge_temp_data;
drop table pharmacy_maplarge_temp_data;

-- Include the ncpdpid from the pharmacy table to uniquely identify the pharmacy
update pharmacy_maplarge_location set ncpdpid = pharmacy.ncpdpid from pharmacy where pharmacy.id = pharmacy_maplarge_location.id;


-- Import results from ersi (this is just a subset of the results for points that we were unable to find high precision data up until this point)
create table pharmacy_ersi_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	display_latitude numeric(10,6),
	disply_longitude numeric(10,6),
	score numeric(10,6)
);
\copy pharmacy_ersi_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/low-precision-ersi-results.csv' WITH DELIMITER ',';

alter table pharmacy_ersi_location add column ncpdpid integer;
update pharmacy_ersi_location set ncpdpid = pharmacy.ncpdpid from pharmacy where pharmacy.id = pharmacy_ersi_location.id;

-- Create table to store all high precision data into
create table pharmacy_location (
	id integer references pharmacy(id),
	ncpdpid integer,
	latitude numeric(10,6) not null,
	longitude numeric(10,6) not null,
	source text not null,
	precision text
);

alter table pharmacy_location add constraint unique_ncpdpid unique (ncpdpid);

-- Insert high precision data from smarty streets
insert into pharmacy_location (id, latitude, longitude, source, precision, ncpdpid) select id, latitude, longitude, 'smarty_streets', 'zip9', ncpdpid from pharmacy_ss_location where zip_precision = 'Zip9';

-- Insert high precision data from maplarge
insert into pharmacy_location (id, latitude, longitude, source, precision, ncpdpid) 
	select id, latitude, longitude, 'maplarge', 'exactmatch,exact', ncpdpid from pharmacy_maplarge_location where matchtype='ExactMatch' and numtype='Exact' 
	and ncpdpid in (select ncpdpid from pharmacy_ss_location where zip_precision != 'Zip9');

-- Insert whatever we can from esri data
insert into pharmacy_location (id, latitude, longitude, source, precision, ncpdpid) select distinct id, latitude, longitude, 'ersi', score, ncpdpid from pharmacy_ersi_location where ncpdpid in 
	(select pharmacy_ss_location.ncpdpid from pharmacy_ss_location, pharmacy_maplarge_location  where pharmacy_ss_location.zip_precision != 'Zip9' 
		and (pharmacy_maplarge_location.matchtype != 'ExactMatch' or pharmacy_maplarge_location.numtype != 'Exact') 
		and pharmacy_ss_location.ncpdpid = pharmacy_maplarge_location.ncpdpid);

-- Insert remaining from smarty streets
insert into pharmacy_location (id, latitude, longitude, source, precision, ncpdpid) 
	select id, latitude, longitude, 'smarty_streets', zip_precision, ncpdpid from pharmacy_ss_location 
	where ncpdpid in (select distinct ncpdpid from pharmacy except (select ncpdpid from pharmacy_location));

-- Create postgis related data types
create extension postgis;
SELECT AddGeometryColumn('public', 'pharmacy_location', 'geom', 4326, 'POINT', 2);
CREATE INDEX pharmacy_location_index ON pharmacy_location USING GIST (geom);
UPDATE pharmacy_location SET geom = ST_GeomFromText('POINT(' || longitude || ' ' || latitude || ')',4326);



