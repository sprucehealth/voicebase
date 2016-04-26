CREATE TABLE deploy.deployable_group ( 
    id                   bigint UNSIGNED NOT NULL,
    name                 varchar(150),
    description          varchar(150),
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE (name),
    CONSTRAINT pk_deployable_group PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.deployable ( 
    id                   bigint UNSIGNED NOT NULL,
    deployable_group_id  bigint UNSIGNED NOT NULL,
    name                 varchar(150),
    description          varchar(150),
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_deployable_deployable_group_id FOREIGN KEY (deployable_group_id) REFERENCES deploy.deployable_group(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
	UNIQUE (deployable_group_id, name),
    CONSTRAINT pk_deployable PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.environment ( 
    id                   bigint UNSIGNED NOT NULL,
    deployable_group_id  bigint UNSIGNED NOT NULL,
    name                 varchar(150),
    description          varchar(150),
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_prod              BOOL NOT NULL,
    CONSTRAINT fk_environment_deployable_group_id FOREIGN KEY (deployable_group_id) REFERENCES deploy.deployable_group(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
	UNIQUE (deployable_group_id, name),
    CONSTRAINT pk_environment PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.deployable_config (
    id                   bigint UNSIGNED NOT NULL,
    deployable_id        bigint UNSIGNED NOT NULL,
    environment_id       bigint UNSIGNED NOT NULL,
    status               varchar(25) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_deployable_config_deployable_id FOREIGN KEY (deployable_id) REFERENCES deploy.deployable(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployable_config_environment_id FOREIGN KEY (environment_id) REFERENCES deploy.environment(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_deployable_config PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.deployable_config_value (
    deployable_config_id bigint UNSIGNED NOT NULL,
    name                 varchar(100) NOT NULL,
    value                varchar(255) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_deployable_config_value PRIMARY KEY (deployable_config_id, name)
) engine=InnoDB;

CREATE TABLE deploy.environment_config (
    id                   bigint UNSIGNED NOT NULL,
    environment_id       bigint UNSIGNED NOT NULL,
    status               varchar(25) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_environment_config_environment_id FOREIGN KEY (environment_id) REFERENCES deploy.environment(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_environment_config PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.environment_config_value (
    environment_config_id bigint UNSIGNED NOT NULL,
    name                  varchar(100) NOT NULL,
    value                 varchar(255) NOT NULL,
    created               timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_environment_config_value PRIMARY KEY (environment_config_id, name)
) engine=InnoDB;

CREATE TABLE deploy.deployable_vector (
    id                    bigint UNSIGNED NOT NULL,
    deployable_id         bigint UNSIGNED NOT NULL,
    source_type           varchar(25) NOT NULL,
    source_environment_id bigint UNSIGNED,
    target_environment_id bigint UNSIGNED NOT NULL,
    created               timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_deployable_vector_deployable_id FOREIGN KEY (deployable_id) REFERENCES deploy.deployable(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployable_vector_source_environment_id FOREIGN KEY (source_environment_id) REFERENCES deploy.environment(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployable_vector_target_environment_id FOREIGN KEY (target_environment_id) REFERENCES deploy.environment(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT pk_deployable_vector PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE deploy.deployment (
    id                    bigint UNSIGNED NOT NULL,
	deployment_number     int UNSIGNED NOT NULL AUTO_INCREMENT,
    environment_id        bigint UNSIGNED NOT NULL,
    deployable_id         bigint UNSIGNED NOT NULL,
    deployable_config_id  bigint UNSIGNED NOT NULL,
    deployable_vector_id  bigint UNSIGNED NOT NULL,
    type                  varchar(25) NOT NULL,
    data                  BLOB NOT NULL,
    status                varchar(25) NOT NULL,
    build_number          varchar(255),
    created               timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_deployment_deployable_id FOREIGN KEY (deployable_id) REFERENCES deploy.deployable(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployment_environment_id FOREIGN KEY (environment_id) REFERENCES deploy.environment(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployment_deployable_config_id FOREIGN KEY (deployable_config_id) REFERENCES deploy.deployable_config(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    CONSTRAINT fk_deployment_deployable_vector_id FOREIGN KEY (deployable_vector_id) REFERENCES deploy.deployable_vector(id) ON DELETE NO ACTION ON UPDATE NO ACTION,
    KEY k_deployment_number (deployment_number), -- Required for auto increment
    CONSTRAINT pk_deployment PRIMARY KEY (id)
) engine=InnoDB;

--TODO: Add triggers to assert existance and singularity of deployable and environment configs