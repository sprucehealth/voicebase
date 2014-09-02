ALTER TABLE care_provider_profile ADD COLUMN undergraduate_school TEXT;
ALTER TABLE care_provider_profile ADD COLUMN graduate_school TEXT;

UPDATE care_provider_profile SET graduate_school = '', undergraduate_school = '';

ALTER TABLE care_provider_profile MODIFY undergraduate_school TEXT NOT NULL;
ALTER TABLE care_provider_profile MODIFY graduate_school TEXT NOT NULL;
