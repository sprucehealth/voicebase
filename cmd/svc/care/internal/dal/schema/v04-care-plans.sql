CREATE TABLE care_plan (
    id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    instructions_json BLOB, -- JSON encoded for easier DML if we switch to the native JSON type in MySQL 5.7
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creator_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    parent_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin,
    submitted TIMESTAMP,
    PRIMARY KEY (id),
    KEY parent_id (parent_id)
);

CREATE TABLE care_plan_treatment (
    id BIGINT UNSIGNED NOT NULL,
    care_plan_id BIGINT UNSIGNED NOT NULL,
    medication_id VARCHAR(256) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    eprescribe BOOLEAN NOT NULL,
    name VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    form VARCHAR(64) NOT NULL,
    route VARCHAR(64) NOT NULL,
    availability VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    dosage VARCHAR(256) NOT NULL,
    dispense_type VARCHAR(256) NOT NULL,
    dispense_number INT NOT NULL,
    refills INT NOT NULL,
    substitutions_allowed BOOLEAN NOT NULL,
    days_supply INT NOT NULL,
    sig TEXT NOT NULL,
    pharmacy_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    pharmacy_instructions TEXT NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT care_plan_id FOREIGN KEY (care_plan_id) REFERENCES care_plan (id)
);
