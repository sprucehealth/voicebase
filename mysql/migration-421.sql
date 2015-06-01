-- A primary CC is chosen randomly to be the CC assigned to a new patient's
-- care team.
--
-- This isn't totally appropriate in the doctor table (which is really a
-- care provider table since it's both doctors and CC), but I don't think
-- it's necessary to add another table for just one bit.
ALTER TABLE doctor ADD COLUMN primary_cc BOOL NOT NULL DEFAULT false;

UPDATE doctor SET primary_cc = true
WHERE account_id IN (
        SELECT id
        FROM account
        WHERE role_type_id = (
            SELECT id FROM role_type WHERE role_type_tag = 'MA'));
