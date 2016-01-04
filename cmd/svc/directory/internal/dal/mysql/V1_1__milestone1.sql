CREATE TABLE directory.entity (
    id                   bigint UNSIGNED NOT NULL,
    name                 varchar(250) NOT NULL,
    type                 varchar(100) NOT NULL,
    status               varchar(50) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX      idx_status (status),
    INDEX      idx_entity_type (type),
    CONSTRAINT pk_entity PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE directory.external_entity_id (
    entity_id            bigint UNSIGNED NOT NULL,
    external_id          varchar(100) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT pk_external_entity_id PRIMARY KEY (entity_id, external_id),
    CONSTRAINT fk_external_entity_id_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION
) engine=InnoDB;
 
CREATE TABLE directory.entity_membership (
    entity_id            bigint UNSIGNED NOT NULL,
    target_entity_id     bigint UNSIGNED NOT NULL,
    status               varchar(100) NOT NULL,
    created              timestamp DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX      idx_status (status),
    CONSTRAINT fk_entity_membership_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_entity_membership_target_entity_id FOREIGN KEY (target_entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_entity_membership PRIMARY KEY (entity_id, target_entity_id)
) engine=InnoDB;

CREATE TABLE directory.entity_contact (
    id                   bigint UNSIGNED NOT NULL,
    entity_id            bigint UNSIGNED NOT NULL,
    type                 varchar(100) NOT NULL,
    value                varchar(100) NOT NULL,
    provisioned             BOOL NOT NULL DEFAULT FALSE,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX      idx_entity_id (entity_id),
    CONSTRAINT fk_entity_contact_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_entity_contact PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE directory.event (
    id                   bigint UNSIGNED NOT NULL,
    entity_id            bigint UNSIGNED NOT NULL,
    event                text NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_entity_event_entity_id FOREIGN KEY (entity_id) REFERENCES directory.entity(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_entity_event PRIMARY KEY (id)
) engine=InnoDB;