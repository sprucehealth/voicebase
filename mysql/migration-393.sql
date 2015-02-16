-- Add an index for efficient lookup based on patient and status of case
ALTER TABLE patient_case ADD KEY patient_id_status_key (patient_id, status);

-- Drop the key with just the patient_id now that we have the composite index above
ALTER TABLE patient_case DROP KEY `patient_id`;

-- Add a bool for bookkeeping on whether a case has been claimed or not. Note that a case is considered
-- claimed if it has a doctor permanently assigned to it.
ALTER TABLE patient_case ADD COLUMN claimed TINYINT(1) NOT NULL DEFAULT 0;

-- Mark a case as being claimed if its indicated so via the status
UPDATE patient_case SET claimed = 1
	WHERE status='CLAIMED';

-- Mark a case as being open if no visit has been submitted
UPDATE patient_case as pc
	INNER JOIN patient_visit AS pv ON pv.patient_case_id = pc.id
	SET pc.status = 'OPEN'
	WHERE pv.status = 'OPEN' and pv.followup = false;

-- Mark a case as being active if the initial visit has been submitted 
UPDATE patient_case as pc
	INNER JOIN patient_visit AS pv ON pv.patient_case_id = pc.id
	SET pc.status = 'ACTIVE'
	WHERE pv.status not in ('OPEN', 'UNSUITABLE', 'PENDING') and pv.followup = false;

