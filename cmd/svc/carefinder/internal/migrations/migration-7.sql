-- To make querying easier, add the spruce doctors to each city in the short list
-- and mark them as spruce doctors
ALTER TABLE doctor_city_short_list ADD COLUMN is_spruce_doctor BOOL NOT NULL DEFAULT FALSE;

CREATE INDEX doctor_city_short_list_key ON doctor_city_short_list USING btree (doctor_id, city_id, is_spruce_doctor);
CREATE INDEX doctor_city_short_list_key_2 ON doctor_city_short_list USING btree (doctor_id, city_id);

INSERT INTO doctor_city_short_list (doctor_id, city_id, npi, is_spruce_doctor) 
	SELECT carefinder_doctor_info.id, city_shortlist.city_id, carefinder_doctor_info.npi, true
	FROM doctor_short_list
	INNER JOIN carefinder_doctor_info ON carefinder_doctor_info.id = doctor_short_list.doctor_id
	INNER JOIN spruce_doctor_state_coverage ON spruce_doctor_state_coverage.doctor_id = doctor_short_list.doctor_id
	INNER JOIN city_shortlist ON city_shortlist.state = spruce_doctor_state_coverage.state_abbreviation
	WHERE doctor_short_list.is_spruce_doctor = true;
 
