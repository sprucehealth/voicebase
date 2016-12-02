-- Track where an entity came from
ALTER TABLE entity ADD COLUMN source VARCHAR(100) NOT NULL DEFAULT ''; 