
-- don't consider email as verified
UPDATE entity_contact
SET verified = 1
WHERE type='email';
