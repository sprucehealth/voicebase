-- Note. The data in the tables here have been manually loaded from various sources via various
-- methods of heuristics and calculations. It is best to restore the data from a snapshot
-- for the carefinder project.

--
-- INDEX CITY AND STATE INFORMATION TO MAKE IT EASILY ACCESSIBLE
--

-- create a human readable key for the cities database which is cityname-state
ALTER TABLE cities ADD COLUMN id TEXT;

-- Only create the key for cities that are registered as cities based on the feature code
-- and have a population > 5000. This is so that we can uniquely identify the cities for which
-- to display content.
UPDATE cities
SET id = translate(lower(cities.name),' ', '-') || '-' || translate(lower(cities.admin1_code), ' ', '-')
WHERE feature_code = 'PPL'
AND population > 5000;

-- Include cities that we have missed out on because they didn't have feature_code specified as PPL but was like PPL
UPDATE cities
SET id = translate(lower(cities.name),' ', '-') || '-' || translate(lower(cities.admin1_code), ' ', '-')
WHERE feature_code like '%PPL%' AND feature_code != 'PPL'
AND population > 5000;

-- Ensure that we can uniquely identify cities. 
-- Note that to create this index I had to manually clean up the duplicate entries (there were 43 cities with
-- the same id) based on data that was latest (based on the last-modified column)
-- and in situations where the last-modified matched, then based on which entry had the larger population
-- taking the assumption that the population of the world is increasing.
CREATE UNIQUE INDEX cities_id_key ON cities USING btree (id);

-- Create a state table
CREATE TABLE state (
	ID INT NOT NULL PRIMARY KEY,
	full_name TEXT NOT NULL,
	abbreviation TEXT NOT NULL,
	country TEXT NOT NULL
);

CREATE UNIQUE INDEX state_abbreviation_key ON state USING btree (abbreviation);

--
-- BUILD A CITY SHORTLIST TO REPRESENT THE CITY PAGES WE WILL SUPPORT FOR CAREFINDER
--

-- city_shortlist will store a shortlist of cities for which we will support 
-- the carefinder city pages.
CREATE TABLE city_shortlist (
	city_id TEXT NOT NULL REFERENCES cities(id)
);

--
-- STORE MAJORITY OF THE DOCTOR INFORMATION IN A SINGLE TABLE FOR QUICK ACCESS
--

-- create a single place to reference most, if not all information required to show a doctor
-- card or doctor profile on the carefinder page.
CREATE TABLE carefinder_doctor_info (
	npi TEXT NOT NULL PRIMARY KEY,
	id TEXT,
	first_name TEXT,
	last_name TEXT,
	profile_image_url TEXT,
	description TEXT,
	is_spruce_doctor BOOL DEFAULT FALSE,
	graduation_year TEXT,
	medical_school TEXT,
	residency TEXT,
	yelp_url TEXT,
	yelp_business_id TEXT,
	average_rating DOUBLE PRECISION,
	insurance_accepted_csv TEXT,
	specialties_csv TEXT,
	practice_location_address_line_1 TEXT,
	practice_location_address_line_2 TEXT,
	practice_location_city TEXT,
	practice_location_state TEXT,
	practice_location_zipcode TEXT,
	practice_location_lat numeric(10,6),
	practice_location_lng numeric(10,6),
	practice_phone TEXT
);

-- rename the profile_image_url to profile_image_id in the carefinder_doctor_info table
-- given that we are storing ids and not actual urls.
ALTER TABLE carefinder_doctor_info RENAME profile_image_url TO profile_image_id;

-- Include gender in the carefinder_doctor_info given that it can be useful
ALTER TABLE carefinder_doctor_info ADD COLUMN gender CHAR(1);

-- Create an id for all doctors which is essentially md-firstname-lastname
UPDATE carefinder_doctor_info
SET id = 'md-' || translate(lower(first_name),' ', '-') || '-' || translate(lower(last_name), ' ', '-');

-- Make the id non nullable in the carefinder-doctor-info table
ALTER TABLE carefinder_doctor_info ALTER COLUMN id SET NOT NULL;

-- the ids for doctors will be hand created and required to be unique. The id 
-- will be the path of the doctor in the URL.
CREATE UNIQUE INDEX doctor_id_key ON carefinder_doctor_info USING btree (id);

-- CREATE A TABLE TO track spruce reviews
CREATE TABLE spruce_review (
	doctor_id TEXT NOT NULL REFERENCES carefinder_doctor_info(id),
	review TEXT NOT NULL
);

--
-- BUILD DOCTOR SHORT LISTS (BOTH FOR LOCAL AND SPRUCE) TO MAKE REPRESENT LIST OF DOCTORS LIVE ON CAREFINDER
--

-- store the list of doctors to include for each city,state combination
CREATE TABLE doctor_short_list (	
	city_id TEXT NOT NULL REFERENCES cities(id),
	npi TEXT NOT NULL
);

-- make for easy lookup using the doctor_id in the doctor_short_list.
ALTER TABLE doctor_short_list ADD COLUMN doctor_id TEXT REFERENCES carefinder_doctor_info(id);

UPDATE doctor_short_list
SET doctor_id = carefinder_doctor_info.id
FROM carefinder_doctor_info
WHERE carefinder_doctor_info.npi = doctor_short_list.npi;

ALTER TABLE doctor_short_list ALTER COLUMN doctor_id SET NOT NULL;

-- create a table to store a short list of the spruce doctor that are actually live on carefinder 
CREATE TABLE spruce_doctor_short_list (
	doctor_id text not null REFERENCES carefinder_doctor_info(id)
);

-- independently store the listing of which state each spruce doctor is registered in
-- so that we can easily add/remove state registrations.
CREATE TABLE spruce_doctor_state_coverage (
	npi TEXT NOT NULL,
	state_abbreviation TEXT NOT NULL REFERENCES state(abbreviation)
);


--
-- STORE ANCILLARY INFORMATION FOR QUICK ACCESS
-- 


-- create a materialized view for the top skin conditions so that its easy to query the snapshot of results
CREATE MATERIALIZED VIEW top_skin_conditions_by_state AS
	SELECT bucket, state.abbreviation as state, count(bucket) as count
	FROM namcs 
	INNER JOIN state ON state.full_name = namcs.state
	WHERE bucket != '' AND rule_out = false 
	GROUP BY bucket, state.abbreviation ORDER BY state.abbreviation, count(bucket) DESC;

-- given that we will be indexing this view by the state, lets create an index on it
CREATE INDEX state_key ON top_skin_conditions_by_state USING btree (state);

-- create a table to store the care rating for each city/state combination
CREATE TABLE care_rating (
	city_id TEXT NOT NULL REFERENCES cities(id),
	score DOUBLE PRECISION NOT NULL,
	grade TEXT not null
);

-- Store the possible images that can be used for a doctor in this table
-- so that we have a backup
CREATE TABLE doctor_image_profile (
	npi TEXT NOT NULL REFERENCES npi(npi),
	image_1_id TEXT,
	image_2_id TEXT,
	image_3_id TEXT,
	image_4_id TEXT
);

CREATE TABLE banner_image (
	state TEXT NOT NULL REFERENCES state(abbreviation),
	city_id TEXT,
	image_id TEXT NOT NULL
);

-- Once images have been added and empty strings have been set to null for city_id, add the foreign key constraint
ALTER TABLE banner_image ADD CONSTRAINT city_id_key FOREIGN KEY (city_id) REFERENCES cities(id);