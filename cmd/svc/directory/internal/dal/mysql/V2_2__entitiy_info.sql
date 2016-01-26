-- Allow an entity to have a note associated with it
ALTER TABLE entity
CHANGE COLUMN name display_name VARCHAR(250) NOT NULL,
ADD COLUMN first_name VARCHAR(250) NOT NULL,
ADD COLUMN middle_initial VARCHAR(1) NOT NULL,
ADD COLUMN last_name VARCHAR(250) NOT NULL,
ADD COLUMN group_name VARCHAR(250) NOT NULL,
ADD COLUMN note TEXT NOT NULL;

-- Allow an entity contact to have a label associated with it
ALTER TABLE entity_contact ADD COLUMN label VARCHAR(255) NOT NULL;