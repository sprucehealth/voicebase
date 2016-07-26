ALTER TABLE incoming_call ADD COLUMN completed TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE incoming_call ADD COLUMN answered_time TIMESTAMP;
ALTER TABLE incoming_call ADD COLUMN completed_time TIMESTAMP;
