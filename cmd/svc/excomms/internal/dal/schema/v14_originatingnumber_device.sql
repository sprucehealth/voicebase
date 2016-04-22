ALTER TABLE originating_phone_number ADD COLUMN device_id VARCHAR(128);
ALTER TABLE originating_phone_number DROP PRIMARY KEY;
UPDATE originating_phone_number SET device_id='' WHERE device_id is NULL;
ALTER TABLE originating_phone_number ADD PRIMARY KEY (entity_id, device_id);