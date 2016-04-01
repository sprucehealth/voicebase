-- Add support for gender
ALTER TABLE entity ADD COLUMN gender VARCHAR(10);

-- Add support for dob
ALTER TABLE entity ADD COLUMN dob DATE;