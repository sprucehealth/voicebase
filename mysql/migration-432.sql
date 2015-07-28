-- Evidence of parent or guardian identity
CREATE TABLE parent_consent_proof (
    patient_id INT UNSIGNED NOT NULL,
    governmentid_media_id INT UNSIGNED,
    selfie_media_id INT UNSIGNED,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (patient_id),
    CONSTRAINT governmentid_media FOREIGN KEY (governmentid_media_id) REFERENCES media (id),
    CONSTRAINT selfie_media FOREIGN KEY (selfie_media_id) REFERENCES media (id),
    CONSTRAINT patient_id FOREIGN KEY (patient_id) REFERENCES patient(id)
);