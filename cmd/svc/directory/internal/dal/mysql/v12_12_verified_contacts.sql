ALTER TABLE entity_contact ADD COLUMN verified TINYINT(1) NOT NULL DEFAULT 0;

-- mark all provisioned phone numbers and email addresses as verified
UPDATE entity_contact
SET verified = 1
WHERE provisioned= 1;

START TRANSACTION;


CREATE TEMPORARY TABLE IF NOT EXISTS entity_ids AS (SELECT id FROM entity WHERE type IN ('PATIENT', 'INTERNAL'));


-- mark phone numbers as verified for patients and providers because we verify phone number for both 
UPDATE entity_contact
SET verified = 1
WHERE entity_id IN (SELECT id from entity_ids)
AND type = 'phone';

COMMIT;


-- emails are only considered verified for patients that came in via invite
-- TODO: via script.
