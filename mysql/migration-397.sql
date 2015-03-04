-- Add tag column and backfill it with normalized titles
ALTER TABLE resource_guide
ADD COLUMN tag varchar(100);
UPDATE resource_guide 
SET tag = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(LOWER(title),' ','_'),',',''),':',''),'-','_'),'\'','');

-- Clean up resource guides that shouldn't be there anymore
DELETE FROM resource_guide WHERE tag in ('','add_me');

-- Make the column not null and add a unique contraint on it
ALTER TABLE resource_guide
MODIFY COLUMN tag varchar(100) NOT NULL;
ALTER TABLE resource_guide ADD CONSTRAINT resource_guide_tag UNIQUE (tag);
