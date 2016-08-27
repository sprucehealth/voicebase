-- Track transaction IDs for payments
ALTER TABLE payment 
ADD COLUMN processor_transaction_id VARCHAR(150) DEFAULT '',
ADD COLUMN processor_status_message VARCHAR(255) DEFAULT '';
ALTER TABLE payment 
MODIFY COLUMN processor_transaction_id VARCHAR(150),
MODIFY COLUMN processor_status_message VARCHAR(150);