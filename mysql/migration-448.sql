-- Add columns to the treatment plan scheduled message table for easy bookkeeping
-- of whether or not a scheduled message has been cancelled or sent.
ALTER TABLE treatment_plan_scheduled_message ADD COLUMN cancelled TINYINT(1) NOT NULL DEFAULT FALSE;
ALTER TABLE treatment_plan_scheduled_message ADD COLUMN cancelled_time TIMESTAMP;
ALTER TABLE treatment_plan_scheduled_message ADD COLUMN sent_time TIMESTAMP NULL;