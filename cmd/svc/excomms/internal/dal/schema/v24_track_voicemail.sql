ALTER TABLE incoming_call ADD COLUMN left_voicemail TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE incoming_call ADD COLUMN left_voicemail_time TIMESTAMP;
