#### Following points to note:

* Using postgressql for the pharmacy database as the postGIS extension is pretty robust and provides for much better spatial database support
* It's acceptable for us to host the postgres instance on RDS because there is no ePHI in the data and we will only be using the database to read pharmacies around locations
* Currently geocoding data using the smartystreets API which is an address validation service that geocodes addresses based on the USPS database
* This article gives a breakdown on the precision of the geocoded data from smarty streets: http://smartystreets.com/kb/faq/how-accurate-is-your-geocoding-data
* Dosespot provided the pharmacy datadump in a csv file that we load into a postgres database


#### Postgres schema 

##### Create the postgres schema based on the csv
```postgres
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
```

##### Create extension for PostGIS
```postgres
create extension postgis
```

Confirm that postgis was successfully installed by running the following command:
```postgres
select postgis_full_version();
```

##### Copy data from file into table
```postgres
COPY pharmacy FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/pharmacy.csv' WITH DELIMITER  ','  CSV HEADER; 
```

##### Create holding table for pharmacy location data
<i> Reason for this is so that we can independently geocode data and then feed it into table </i>
```postgres
create table pharmacy_location (
	id integer references pharmacy(id),
	latitude float8,
	longitude float8,
	zip_precision text
);
```

##### Copy geocoded data into table
<i> Note that data is tab-delimited with just the id,latitude,longitude, zip_precision in the file </i>
```postgres
COPY pharmacy_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-first-50000-results-short.txt';
COPY pharmacy_location FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/ListProcessing-Python/surescripts-pharmacy-remaining-results-short.txt';
```

##### Add longitude/latitude columns
```postgres
alter table pharmacy add column latitude float8;
alter table pharmacy add column longitude float8;
alter table pharmacy add column zip_precision text;
```

##### Add geometric column to pharmacy table
<i> Note that SRID 4326 represents a lat/lng pair: http://spatialreference.org/ref/epsg/wgs-84/ </i> <br>
<i> Documentation: http://postgis.refractions.net/docs/using_postgis_dbmanagement.html#geometry_columns  </i>
``` postgres
SELECT AddGeometryColumn('public', 'pharmacy', 'geom', 4326, 'POINT', 2);
```

##### Create GiST index for spatial data access
<i> Documentation: http://postgis.net/docs/manual-2.1/using_postgis_dbmanagement.html#gist_indexes </i>
```postgres
CREATE INDEX pharmacy_index ON pharmacy USING GIST (geom);
```

##### Update pharmacy table to contain lat/lng data from pharmacy_location 
```postgres
UPDATE pharmacy SET longitude = pharmacy_location.longitude, latitude = pharmacy_location.latitude, zip_precision = pharmacy_location.zip_precision
	FROM pharmacy_location where pharmacy_location.id = pharmacy.id; 
```

##### Update geom column to convert the lat/lng into the corresponding geometry type
<i> Documentation: http://www.kevfoo.com/2012/01/Importing-CSV-to-PostGIS/ </i>
```postgres
UPDATE pharmacy SET geom = ST_GeomFromText('POINT(' || longitude || ' ' || latitude || ')',4326);
```

##### Query for data based on provided point and max. radius, ordered by distance
<i> Documentation: http://boundlessgeo.com/2011/09/indexed-nearest-neighbour-search-in-postgis/ </i> <br>
<i> Note: The order by distance with <-> operator is key to making sure we are using the index </i>
```postgres
SELECT id, store_name, address_line_1, city, state, zip from pharmacy
	WHERE st_distance(geom, st_setsrid(st_makepoint(-122.432676,37.781791),4326)) < 16093 
	ORDER BY geom <-> st_setsrid(st_makepoint(-122.349798,47.615579),4326)
	LIMIT 10;
```


