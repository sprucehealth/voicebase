ALTER TABLE incoming_call ADD COLUMN answered TINYINT(1) NOT NULL default 0;
ALTER TABLE incoming_call ADD COLUMN sent_to_voicemail TINYINT(1) NOT NULL default 0;
