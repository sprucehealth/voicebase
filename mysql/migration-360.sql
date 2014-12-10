ALTER TABLE doctor_queue ADD COLUMN description VARCHAR(2000) NOT NULL;
ALTER TABLE doctor_queue ADD COLUMN action_url VARCHAR(2000) NOT NULL;

ALTER TABLE unclaimed_case_queue ADD COLUMN description VARCHAR(2000) NOT NULL;
ALTER TABLE unclaimed_case_queue ADD COLUMN action_url VARCHAR(2000) NOT NULL;

-- Delete deprecated items from the doctor queue
DELETE FROM doctor_queue WHERE event_type='CONVERSATION';
DELETE FROM doctor_queue WHERE status='PHOTOS_REJECTED';
DELETE FROM doctor_queue WHERE status='deny';
DELETE FROM doctor_queue WHERE status='';

-- Add description and actionurl to the unassigned queue
UPDATE unclaimed_case_queue as uq
	INNER JOIN patient_case as pc ON pc.id = uq.patient_case_id
	INNER JOIN patient as p ON p.id = pc.patient_id 
	SET description = CONCAT('New visit with ', p.first_name, ' ', p.last_name),
	action_url = CONCAT('spruce:///action/view_patient_visit?case_id=',pc.id, '&patient_id=' , pc.patient_id ,'&patient_visit_id=',uq.item_id);

