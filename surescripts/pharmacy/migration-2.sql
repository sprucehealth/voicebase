create table pharmacy_maplarge_location (
	id integer references pharmacy(id),
	latitude float8,
	longitude float8,
	matchtype text,
	numtype text
);

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
insert into pharmacy_maplarge_location (id, latitude, longitude, matchtype, numtype) select id, latitude, longitude, matchtype, numtype from pharmacy_maplarge_temp_data;


create table pharmacy_smartystreets_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	zip_precision text
);


\copy pharmacy_smartystreets_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-first-50000-results-short.txt';
\copy pharmacy_smartystreets_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-remaining-results-short.txt';


alter table pharmacy_smartystreets_location add column ncpdpid integer;

update pharmacy_smartystreets_location
 	set ncpdpid = pharmacy.ncpdpid
 	from pharmacy where pharmacy.id = pharmacy_smartystreets_location.id;

create table pharmacy_ss_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	zip_precision text
);

insert into pharmacy_ss_location (id, latitude, longitude, zip_precision) select distinct on (ncpdpid) id, latitude, longitude, zip_precision from pharmacy_smartystreets_location;

drop table pharmacy_smartystreets_location;

create table pharmacy_location (
	id integer references pharmacy(id),
	latitude numeric(10,6) not null,
	longitude numeric(10,6) not null,
	source text not null,
	precision text
);

-- Insert data from smarty streets
insert into pharmacy_location (id, latitude, longitude, source, precision) select distinct on (ncpdpid) id, latitude, longitude, 'smarty_streets', 'zip9' from pharmacy where zip_precision = 'Zip9';

-- Insert exact data from maplarge
insert into pharmacy_location (id, latitude, longitude, source, precision) 
	select id, latitude, longitude, 'maplarge', 'exactmatch,exact' from pharmacy_maplarge_location where matchtype='ExactMatch' and numtype='Exact' and id in (select id from pharmacy_ss_location where zip_precision != 'Zip9');

create table pharmacy_ersi_location (
	id integer references pharmacy(id),
	latitude numeric(10,6),
	longitude numeric(10,6),
	display_latitude numeric(10,6),
	disply_longitude numeric(10,6),
	score numeric(10,6)
);

\copy pharmacy_ersi_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/low-precision-ersi-results.csv' WITH DELIMITER ',';

-- Insert data from ersi (this only is a subset of the data that was geocoded)
insert into pharmacy_location (id, latitude, longitude, source, precision)
	select id, latitude, longitude, 'ersi', score from pharmacy_ersi_location;

-- Insert data for remaining pharmacies
insert into pharmacy_location (id, latitude, longitude, source, precision)
	select id, latitude, longitude, 'smarty_streets', zip_precision from pharmacy_ss_location
	where id in (select distinct on (ncpdpid) id from pharmacy except (select id from pharmacy_location));

SELECT AddGeometryColumn('public', 'pharmacy_location', 'geom', 4326, 'POINT', 2);
CREATE INDEX pharmacy_location_index ON pharmacy_location USING GIST (geom);
UPDATE pharmacy_location SET geom = ST_GeomFromText('POINT(' || longitude || ' ' || latitude || ')',4326);

UPDATE pharmacy_location
	set pharmacy_location.latitude = pharmacy_maplarge_location.latitude,
	pharmacy_location.longitude = pharmacy_maplarge_location.longitude,
	pharmacy_location.precision = 'exactmatch,exact',
	pharmacy_location.source = 'maplarge'
	FROM pharmacy_maplarge_location where pharmacy_maplarge_location.id = pharmacy_location.id 
	AND pharmacy_maplarge_location.matchtype='ExactMatch' AND pharmacy_maplarge_location.numtype='Exact';


SELECT id, ncpdpid, store_name, address_line_1, address_line_2, city, state, zip, phone_primary, fax, pharmacy_location.longitude, pharmacy_location.latitude FROM pharmacy, pharmacy_location
		WHERE  pharmacy.id = pharmacy_location.id
			AND st_distance(geom, st_setsrid(st_makepoint($1,$2),4326)) < $3
			AND mod(service_level, 2) = 1
			ORDER BY geom <-> st_setsrid(st_makepoint($1,$2),4326)
			LIMIT $4
