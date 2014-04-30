CREATE TABLE patient_notifications (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    patient_id INT UNSIGNED NOT NULL,
    uid VARCHAR(128) NOT NULL,
    tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires TIMESTAMP NULL DEFAULT NULL,
    dismissible BOOL NOT NULL,
    dismiss_on_action BOOL NOT NULL,
    priority INT NOT NULL,
    type VARCHAR(64) NOT NULL,
    data BLOB NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patient (id),
    UNIQUE (patient_id, uid),
    PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE health_log (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    patient_id INT UNSIGNED NOT NULL,
    uid VARCHAR(128) NOT NULL,
    tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    type VARCHAR(64) NOT NULL,
    data BLOB NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patient (id),
    UNIQUE (patient_id, uid),
    PRIMARY KEY (id)
) CHARACTER SET utf8;
