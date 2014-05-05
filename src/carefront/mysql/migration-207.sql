
CREATE TABLE person (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    role_type VARCHAR(32) NOT NULL, -- PATIENT, DOCTOR
    role_id INT UNSIGNED NOT NULL,
    UNIQUE (role_type, role_id),
    PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE photo (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    uploaded TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uploader_id BIGINT UNSIGNED NOT NULL,
    status VARCHAR(32) NOT NULL,
    mimetype VARCHAR(128) NOT NULL, -- image/*
    url VARCHAR(255) NOT NULL, -- s3://region/bucket/....
    claimer_type VARCHAR(64) DEFAULT NULL, -- message
    claimer_id BIGINT UNSIGNED DEFAULT NULL, -- message_id
    FOREIGN KEY (uploader_id) REFERENCES person (id),
    PRIMARY KEY (id)
);

CREATE TABLE conversation_topic (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    ordinal INT NOT NULL,
    active BOOL NOT NULL,
    PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE conversation (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    topic_id INT UNSIGNED NOT NULL,
    message_count INT NOT NULL,
    creator_id BIGINT UNSIGNED NOT NULL,
    owner_id BIGINT UNSIGNED NOT NULL, -- who is responsible for the next action
    last_participant_id BIGINT UNSIGNED NOT NULL,
    last_message_tstamp TIMESTAMP NOT NULL,
    unread BOOL NOT NULL,
    FOREIGN KEY (topic_id) REFERENCES conversation_topic (id),
    FOREIGN KEY (creator_id) REFERENCES person (id),
    FOREIGN KEY (owner_id) REFERENCES person (id),
    FOREIGN KEY (last_participant_id) REFERENCES person (id),
    PRIMARY KEY (id)
) CHARACTER SET utf8;

-- Links persons to conversations to be able to efficiently lookup a
-- list of conversations in which an person is a participant.
CREATE TABLE conversation_participant (
    person_id BIGINT UNSIGNED NOT NULL,
    conversation_id BIGINT UNSIGNED NOT NULL,
    joined TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (person_id) REFERENCES person (id),
    FOREIGN KEY (conversation_id) REFERENCES conversation (id),
    PRIMARY KEY (person_id, conversation_id)
) CHARACTER SET utf8;

CREATE TABLE conversation_message (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    conversation_id BIGINT UNSIGNED NOT NULL,
    person_id BIGINT UNSIGNED NOT NULL,
    body TEXT NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversation (id),
    FOREIGN KEY (person_id) REFERENCES person (id),
    KEY (conversation_id, tstamp),
    PRIMARY KEY (id)
) CHARACTER SET utf8;

CREATE TABLE conversation_message_attachment (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    message_id BIGINT UNSIGNED,
    item_type VARCHAR(64) NOT NULL, -- photo, treatment_plan
    item_id BIGINT UNSIGNED NOT NULL,
    FOREIGN KEY (message_id) REFERENCES conversation_message (id),
    PRIMARY KEY (id)
) CHARACTER SET utf8;

INSERT INTO person (role_type, role_id) SELECT 'PATIENT', id FROM patient;
INSERT INTO person (role_type, role_id) SELECT 'DOCTOR', id FROM doctor;

INSERT INTO conversation_topic (title, ordinal, active) VALUES
    ('Acne Treatent Plan', 100, 1),
    ('Prescriptions', 200, 1),
    ('Side Effects', 300, 1),
    ('Not Seeing Results', 400, 1),
    ('Other', 500, 1);
