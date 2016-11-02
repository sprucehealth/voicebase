
-- don't consider email as verified
UPDATE entity_contact
SET verified = 0
WHERE type='email';
