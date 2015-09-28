-- Map and FK the state table into the care_providing _state table
ALTER TABLE care_providing_state 
  ADD COLUMN state_id INT UNSIGNED,
  ADD CONSTRAINT fk_care_providing_state_state_id FOREIGN KEY (state_id) REFERENCES state(id);

UPDATE care_providing_state SET state_id = (SELECT id FROM state WHERE abbreviation = care_providing_state.state);

ALTER TABLE care_providing_state 
  MODIFY COLUMN state_id INT UNSIGNED NOT NULL;

-- Clone the existing practice model record into all states that the doctor is eligible for
ALTER TABLE practice_model
  ADD COLUMN state_id INT UNSIGNED,
  DROP FOREIGN KEY practice_model_doctor_id,
  DROP PRIMARY KEY;
  
INSERT INTO practice_model (doctor_id, spruce_pc, practice_extension, state_id)
SELECT DISTINCT provider_id doctor_id, practice_model.spruce_pc spruce_pc, practice_model.practice_extension practice_extension, (care_providing_state.state_id) state_id
  FROM care_provider_state_elligibility
  JOIN care_providing_state ON care_providing_state.id = care_providing_state_id
  JOIN state ON care_providing_state.state_id = state.id
  JOIN practice_model ON care_provider_state_elligibility.provider_id = practice_model.doctor_id;

-- Purge the old stateless records
DELETE FROM practice_model WHERE state_id IS NULL;

-- Rebuild the table's info
ALTER TABLE practice_model
  ADD PRIMARY KEY (doctor_id, state_id),
  ADD CONSTRAINT fk_practice_model_state_id FOREIGN KEY (state_id) REFERENCES state(id),
  ADD CONSTRAINT fk_practice_model_doctor_id FOREIGN KEY (doctor_id) REFERENCES doctor(id);

-- Add the ability to track the visit type from the patient_case table
ALTER TABLE patient_case
  ADD COLUMN practice_extension BOOL NOT NULL DEFAULT false;