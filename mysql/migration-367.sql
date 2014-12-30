CREATE TABLE treatment_plan_resource_guide (
	treatment_plan_id INT UNSIGNED NOT NULL,
	resource_guide_id INT UNSIGNED NOT NULL,
	primary key (treatment_plan_id, resource_guide_id),
	FOREIGN KEY (treatment_plan_id) REFERENCES treatment_plan (id) ON DELETE CASCADE,
	FOREIGN KEY (resource_guide_id) REFERENCES resource_guide (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE dr_favorite_treatment_plan_resource_guide (
	dr_favorite_treatment_plan_id INT UNSIGNED NOT NULL,
	resource_guide_id INT UNSIGNED NOT NULL,
	primary key (dr_favorite_treatment_plan_id, resource_guide_id),
	FOREIGN KEY (dr_favorite_treatment_plan_id) REFERENCES dr_favorite_treatment_plan (id) ON DELETE CASCADE,
	FOREIGN KEY (resource_guide_id) REFERENCES resource_guide (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- The scheduled message migration had a bug where it didn't
-- create the foreign key references. MySQL apparently doesn't
-- give errors for syntax it doesn't handle and instead thinks
-- it's ok to just ignore it.
--
-- Also, it was referencing the wrong table.. which MySQL also ignored.

RENAME TABLE favorite_treatment_plan_scheduled_message TO dr_favorite_treatment_plan_scheduled_message;
RENAME TABLE favorite_treatment_plan_scheduled_message_attachment TO dr_favorite_treatment_plan_scheduled_message_attachment;

ALTER TABLE media_claim MODIFY COLUMN media_id INT UNSIGNED NOT NULL;
ALTER TABLE media_claim ADD FOREIGN KEY (media_id) REFERENCES media (id) ON DELETE CASCADE;
ALTER TABLE treatment_plan_scheduled_message MODIFY COLUMN treatment_plan_id INT UNSIGNED NOT NULL;
ALTER TABLE treatment_plan_scheduled_message ADD FOREIGN KEY (treatment_plan_id) REFERENCES treatment_plan (id) ON DELETE CASCADE;
ALTER TABLE treatment_plan_scheduled_message_attachment ADD FOREIGN KEY (treatment_plan_scheduled_message_id) REFERENCES treatment_plan_scheduled_message (id) ON DELETE CASCADE;
ALTER TABLE dr_favorite_treatment_plan_scheduled_message CHANGE COLUMN favorite_treatment_plan_id dr_favorite_treatment_plan_id INT UNSIGNED NOT NULL;
ALTER TABLE dr_favorite_treatment_plan_scheduled_message ADD FOREIGN KEY (dr_favorite_treatment_plan_id) REFERENCES dr_favorite_treatment_plan (id) ON DELETE CASCADE;
ALTER TABLE dr_favorite_treatment_plan_scheduled_message_attachment CHANGE COLUMN favorite_treatment_plan_scheduled_message_id dr_favorite_treatment_plan_scheduled_message_id BIGINT UNSIGNED NOT NULL;
ALTER TABLE dr_favorite_treatment_plan_scheduled_message_attachment ADD FOREIGN KEY (dr_favorite_treatment_plan_scheduled_message_id) REFERENCES dr_favorite_treatment_plan_scheduled_message (id) ON DELETE CASCADE;
