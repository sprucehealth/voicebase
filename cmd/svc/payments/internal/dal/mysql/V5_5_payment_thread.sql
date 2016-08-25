-- Track the thread id associated with a payment if there is one
ALTER TABLE payment ADD COLUMN thread_id VARCHAR(150) DEFAULT '';
-- Remove the backfill default
ALTER TABLE payment MODIFY COLUMN thread_id VARCHAR(150);