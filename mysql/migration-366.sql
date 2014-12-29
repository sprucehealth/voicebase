-- Allow multiple claims against the same media. This is necessary to
-- allow reusing the media for scheduled treatment plan messages.

CREATE TABLE media_claim (
    media_id BIGINT UNSIGNED NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    claimer_type VARCHAR(64) DEFAULT NULL,
    claimer_id BIGINT UNSIGNED DEFAULT NULL,
    UNIQUE (media_id, claimer_type, claimer_id),
    KEY (claimer_type, claimer_id)
);

INSERT INTO media_claim (media_id, claimer_type, claimer_id)
    SELECT id, claimer_type, claimer_id FROM media;
ALTER TABLE media DROP COLUMN claimer_type;
ALTER TABLE media DROP COLUMN claimer_id;


-- Add missing index on scheduled_message and drop unused column available_after
CREATE INDEX scheduled_message_status_scheduled ON scheduled_message (status, scheduled);
ALTER TABLE scheduled_message DROP COLUMN available_after;

CREATE TABLE treatment_plan_scheduled_message (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    treatment_plan_id BIGINT UNSIGNED NOT NULL REFERENCES treatment_plan (id),
    scheduled_days INT UNSIGNED NOT NULL,
    message TEXT NOT NULL,
    scheduled_message_id BIGINT NULL REFERENCES scheduled_message (id) ON DELETE SET NULL,
    PRIMARY KEY (id)
);

CREATE TABLE treatment_plan_scheduled_message_attachment (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    treatment_plan_scheduled_message_id BIGINT unsigned NOT NULL REFERENCES treatment_plan_scheduled_message (id) ON DELETE CASCADE,
    item_type VARCHAR(64) NOT NULL,
    item_id BIGINT unsigned NOT NULL,
    title VARCHAR(256) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE favorite_treatment_plan_scheduled_message (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    favorite_treatment_plan_id BIGINT UNSIGNED NOT NULL REFERENCES favorite_treatment_plan (id) ON DELETE CASCADE,
    scheduled_days INT UNSIGNED NOT NULL,
    message TEXT NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE favorite_treatment_plan_scheduled_message_attachment (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    favorite_treatment_plan_scheduled_message_id BIGINT unsigned NOT NULL REFERENCES favorite_treatment_plan_scheduled_message (id) ON DELETE CASCADE,
    item_type VARCHAR(64) NOT NULL,
    item_id BIGINT unsigned NOT NULL,
    title VARCHAR(256) NOT NULL,
    PRIMARY KEY (id)
);
