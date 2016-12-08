ALTER TABLE proxy_phone_number_reservation ADD COLUMN provisioned_phone_number VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin;
ALTER TABLE provisioned_endpoint ADD COLUMN uuid VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin;
CREATE UNIQUE INDEX provisioned_endpoint_uuid ON provisioned_endpoint (uuid);
