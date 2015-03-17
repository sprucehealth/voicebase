-- Update the state of patient cases for any unlinked patient to be inactive
UPDATE patient_case as pc 
INNER JOIN patient ON patient.id = pc.patient_id
SET pc.status = 'INACTIVE'
WHERE patient.status = 'UNLINKED'
AND pc.status = 'OPEN';
