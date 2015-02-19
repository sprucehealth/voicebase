-- The index provider_role_id is not needed because index eligible_doctor_lookup
-- include the role as the first column.
ALTER TABLE care_provider_state_elligibility DROP INDEX provider_role_id;

-- Want to be able to efficiently list elligibility by state
ALTER TABLE care_providing_state ADD KEY state (state);
