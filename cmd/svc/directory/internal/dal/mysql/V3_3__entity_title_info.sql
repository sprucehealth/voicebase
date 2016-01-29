-- Add title information into the entity table
ALTER TABLE entity
ADD COLUMN short_title VARCHAR(250) NOT NULL,
ADD COLUMN long_title VARCHAR(250) NOT NULL;