ALTER TABLE proxy_phone_number_reservation ADD COLUMN provisioned_phone_number VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin;
ALTER TABLE proxy_phone_number_reservation ADD COLUMN uuid VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin;
CREATE UNIQUE INDEX proxy_phone_number_reservation_uuid ON proxy_phone_number_reservation (uuid);
