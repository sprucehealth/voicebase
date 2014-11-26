-- Patient treatment plan regimen tables

CREATE TABLE `regimen_section` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `title` VARCHAR(500) NOT NULL,
    `creation_date` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `treatment_plan_id` int(10) unsigned NOT NULL,
    PRIMARY KEY (`id`),
    KEY `treatment_plan_id` (`treatment_plan_id`),
    FOREIGN KEY (`treatment_plan_id`) REFERENCES `treatment_plan` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO regimen_section (title, treatment_plan_id, creation_date)
    SELECT regimen_type, treatment_plan_id, MIN(creation_date)
    FROM regimen
    GROUP BY treatment_plan_id, regimen_type;

ALTER TABLE regimen ADD COLUMN regimen_section_id INT UNSIGNED NOT NULL REFERENCES regimen_section (id);

UPDATE regimen SET regimen_section_id = (
    SELECT id
    FROM regimen_section rs
    WHERE rs.treatment_plan_id = regimen.treatment_plan_id
        AND regimen_type = rs.title);

UPDATE regimen_section SET
    title = CASE title
    WHEN 'morning' THEN 'Morning'
    WHEN 'night' THEN 'Night'
    WHEN 'spot' THEN 'Spot'
    WHEN 'Nighttime' THEN 'Night'
    WHEN 'As Needed' THEN 'As Needed'
    WHEN 'Daily' THEN 'Daily'
    ELSE title
    END;

ALTER TABLE regimen DROP COLUMN regimen_type;

-- Favorite treatment plan regimen tables

CREATE TABLE `dr_favorite_regimen_section` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `title` VARCHAR(500) NOT NULL,
    `creation_date` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    `dr_favorite_treatment_plan_id` int(10) unsigned NOT NULL,
    PRIMARY KEY (`id`),
    KEY `dr_favorite_treatment_plan_id` (`dr_favorite_treatment_plan_id`),
    FOREIGN KEY (`dr_favorite_treatment_plan_id`) REFERENCES `dr_favorite_treatment_plan` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO dr_favorite_regimen_section (title, dr_favorite_treatment_plan_id, creation_date)
    SELECT regimen_type, dr_favorite_treatment_plan_id, MIN(creation_date)
    FROM dr_favorite_regimen
    GROUP BY dr_favorite_treatment_plan_id, regimen_type;

ALTER TABLE dr_favorite_regimen ADD COLUMN dr_favorite_regimen_section_id INT UNSIGNED NOT NULL REFERENCES dr_favorite_regimen_section (id);

UPDATE dr_favorite_regimen SET dr_favorite_regimen_section_id = (
    SELECT id
    FROM dr_favorite_regimen_section rs
    WHERE rs.dr_favorite_treatment_plan_id = dr_favorite_regimen.dr_favorite_treatment_plan_id
        AND regimen_type = rs.title);

UPDATE dr_favorite_regimen_section SET
    title = CASE title
    WHEN 'morning' THEN 'Morning'
    WHEN 'night' THEN 'Night'
    WHEN 'spot' THEN 'Spot'
    WHEN 'Nighttime' THEN 'Night'
    WHEN 'As Needed' THEN 'As Needed'
    WHEN 'Daily' THEN 'Daily'
    ELSE title
    END;

ALTER TABLE dr_favorite_regimen DROP COLUMN regimen_type;
