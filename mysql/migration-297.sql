CREATE TABLE `patient_exported_medical_record` (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    patient_id INT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL,
    error VARCHAR(256),
    storage_url VARCHAR(512),
    requested_timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_timestamp timestamp NULL DEFAULT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (patient_id) REFERENCES patient (id)
) character set utf8;

-- Unfortunately MySQL is brain dead and doesn't allow not null timestamp columns without a default value.
-- This is required since we enabled strict mode the other alterations below will fail without this default.
ALTER TABLE auth_token MODIFY COLUMN expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE auth_token ADD COLUMN platform VARCHAR(128);
UPDATE auth_token SET platform = 'mobile';
ALTER TABLE auth_token MODIFY COLUMN platform VARCHAR(128) NOT NULL;
DROP INDEX account_id_2 ON auth_token;
CREATE UNIQUE INDEX account_platform ON auth_token (account_id, platform);
