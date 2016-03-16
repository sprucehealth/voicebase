ALTER TABLE provisioned_endpoint ADD COLUMN deprovisioned TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE provisioned_endpoint ADD COLUMN deprovisioned_timestamp TIMESTAMP;
ALTER TABLE provisioned_endpoint ADD COLUMN deprovision_reason VARCHAR(255);