-- Infering the sent date for ACTIVE treatment plans from the last_modified_date
-- given that the last update would be when the status changed from DRAFT -> INACTIVE.
UPDATE treatment_plan
	SET sent_date = last_modified_date
	WHERE status = 'ACTIVE';

-- For the inactive treatment plans, inferring the sent date from when a prescription
-- in the treatment plan was recorded as being sent. 
-- Now this might leave out inactive treatment plans that didn't have any prescriptions,
-- but having looked at the production database its just 2 of the 260 inactive TPs that are left out
-- so I think we're good.
UPDATE treatment_plan as tp
INNER JOIN treatment ON treatment.treatment_plan_id = tp.id
INNER JOIN erx_status_events ON erx_status_events.treatment_id = treatment.id
SET tp.sent_date = erx_status_events.creation_date
WHERE tp.status = 'INACTIVE'
AND erx_status_events.erx_status = 'Sending'
AND tp.sent_date = null;

-- Adding creation_date to the ftp_membership table so that we are capturing when the membership was created.
ALTER TABLE dr_favorite_treatment_plan_membership ADD COLUMN creation_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
