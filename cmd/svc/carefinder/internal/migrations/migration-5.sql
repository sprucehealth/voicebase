-- Add this column so that we can send the provider id to embed in the branch link for attribution purposes
ALTER TABLE carefinder_doctor_info ADD COLUMN spruce_provider_id INTEGER;