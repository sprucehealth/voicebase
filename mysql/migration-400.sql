-- Add a timeout date column to track when a case is meant to timeout of its current status
ALTER TABLE patient_case ADD COLUMN timeout_date timestamp NULL DEFAULT NULL;

-- Update any current pre-submission-triage cases to pre-submission-triage-deleted so that we don't have 
-- to deal with the lack of case notification for the deleted case
UPDATE patient_case SET status = 'PRE_SUBMISSION_TRIAGE_DELETED' where status = 'PRE_SUBMISSION_TRIAGE';

-- Add an index to make lookup by timeout_date efficient
ALTER TABLE patient_case ADD INDEX `timeout_date_index` (timeout_date);


