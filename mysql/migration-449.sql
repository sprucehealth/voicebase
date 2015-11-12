-- ensure that cancelled_time is nullable
ALTER TABLE treatment_plan_scheduled_message MODIFY COLUMN cancelled_time TIMESTAMP NULL;

-- Update all previously sent tretment plan scheduled messages to indicate so
UPDATE treatment_plan_scheduled_message as tpsm
INNER JOIN scheduled_message sm ON sm.id = tpsm.scheduled_message_id
SET sent_time = sm.scheduled
WHERE sm.status='SENT';

-- Update all deactivated messages to indicate that they have been cancelled
UPDATE treatment_plan_scheduled_message as tpsm
INNER JOIN scheduled_message sm ON sm.id = tpsm.scheduled_message_id
SET cancelled = true, cancelled_time = sm.scheduled
WHERE sm.status='DEACTIVATED';