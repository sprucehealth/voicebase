CREATE TABLE directory.entity_profile (
    id                          BIGINT UNSIGNED NOT NULL,
	entity_id                   BIGINT UNSIGNED NOT NULL,
    sections                    BLOB NOT NULL,
    created                     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified                    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY idx_status      (id),
    UNIQUE                      (entity_id),
    CONSTRAINT fk_entity_profile_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;

ALTER TABLE directory.entity ADD COLUMN image_media_id VARCHAR(150) NOT NULL DEFAULT '';