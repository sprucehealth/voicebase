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

create extension postgis;

\copy pharmacy FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/pharmacy.csv' WITH DELIMITER  ','  CSV HEADER;

create table pharmacy_location (
	id integer references pharmacy(id),
	latitude float8,
	longitude float8,
	zip_precision text
);

\copy pharmacy_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-first-50000-results-short.txt';
\copy pharmacy_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-remaining-results-short.txt';

alter table pharmacy add column latitude float8;
alter table pharmacy add column longitude float8;
alter table pharmacy add column zip_precision text;

SELECT AddGeometryColumn('public', 'pharmacy', 'geom', 4326, 'POINT', 2);

CREATE INDEX pharmacy_index ON pharmacy USING GIST (geom);

UPDATE pharmacy SET longitude = pharmacy_location.longitude, latitude = pharmacy_location.latitude, zip_precision = pharmacy_location.zip_precision
	FROM pharmacy_location where pharmacy_location.id = pharmacy.id; 

UPDATE pharmacy SET geom = ST_GeomFromText('POINT(' || longitude || ' ' || latitude || ')',4326);

drop table pharmacy_location;
