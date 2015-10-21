-- Rename the table to more accurately represent what it is
ALTER TABLE doctor_short_list RENAME TO doctor_city_short_list;

-- Rename the spruce_doctor_short_list to be a generic doctor short list
ALTER TABLE spruce_doctor_short_list RENAME TO doctor_short_list;

-- Add a column to indicate whether the doctor short listed is a spruce doctor or not
ALTER TABLE doctor_short_list ADD COLUMN is_spruce_doctor BOOL NOT NULL DEFAULT false;

-- Now set all the doctors in there to indicate they are spruce doctors
UPDATE doctor_short_list
SET is_spruce_doctor = true;

-- Now add the rest of the doctors in there
INSERT INTO doctor_short_list (doctor_id) SELECT distinct doctor_id from doctor_city_short_list;

ALTER TABLE spruce_doctor_state_coverage ADD COLUMN doctor_id TEXT REFERENCES carefinder_doctor_info(id);

UPDATE spruce_doctor_state_coverage
SET doctor_id = carefinder_doctor_info.id
FROM carefinder_doctor_info
WHERE carefinder_doctor_info.npi = spruce_doctor_state_coverage.npi;

ALTER TABLE spruce_doctor_state_coverage ALTER COLUMN doctor_id SET NOT NULL;

CREATE TABLE cropped_images (
	npi TEXT NOT NULL, 
	image_id TEXT NOT NULL
);

\copy cropped_images FROM '/Users/kunaljham/Desktop/uploaded_cropped_images.csv' WITH DELIMITER ',';

UPDATE carefinder_doctor_info
SET profile_image_id = image_id
FROM cropped_images
WHERE cropped_images.npi = carefinder_doctor_info.npi
AND is_spruce_doctor=false;


-- Add state column to city_shortlist;
ALTER TABLE city_shortlist ADD COLUMN state TEXT REFERENCES state(abbreviation);

UPDATE city_shortlist
SET state = admin1_code
FROM cities
WHERE cities.id = city_id;

ALTER TABLE city_shortlist ALTER COLUMN state SET NOT NULL;


ALTER TABLE spruce_review ADD COLUMN created_date DATE;
ALTER TABLE spruce_review ADD COLUMN rating INTEGER;

ALTER TABLE spruce_review ALTER COLUMN created_date SET NOT NULL;
ALTER TABLE spruce_review ALTER COLUMN rating SET NOT NULL;

-- ADD Review count to the carefinder doctor info
ALTER TABLE carefinder_doctor_info ADD COLUMN review_count INTEGER;

UPDATE carefinder_doctor_info 
SET review_count = reviews
FROM yelp_data 
WHERE yelp_data.npi = carefinder_doctor_info.npi;


CREATE TABLE spruce_review_count (
	npi text not null,
	review_count integer not null
);

\copy spruce_review_count FROM '/Users/kunaljham/Desktop/spruce_review_count.tsv' WITH DELIMITER E'\t';

UPDATE carefinder_doctor_info
SET review_count = src.review_count
FROM spruce_review_count AS src
WHERE src.npi = carefinder_doctor_info.npi
AND carefinder_doctor_info.is_spruce_doctor = true;

ALTER TABLE care_rating ADD COLUMN bullets TEXT;
ALTER TABLE care_rating ADD COLUMN title TEXT;


CREATE TABLE care_rating_bullets (
	city_id TEXT NOT NULL REFERENCES cities(id),
	title TEXT NOT NULL,
	bullets TEXT NOT NULL
);


ALTER TABLE care_rating ALTER COLUMN bullets SET NOT NULL;
ALTER TABLE care_rating ALTER COLUMN title SET NOT NULL;

