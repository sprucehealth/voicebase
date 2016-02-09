CREATE TABLE directory.serialized_client_entity_contact (
    entity_id                   bigint UNSIGNED NOT NULL,
    serialized_entity_contact   BLOB NOT NULL,
    platform                    varchar(50) NOT NULL,
    created                     timestamp DEFAULT CURRENT_TIMESTAMP,
    modified                    timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY idx_status (entity_id, platform),
    CONSTRAINT fk_serialized_client_entity_contact_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;