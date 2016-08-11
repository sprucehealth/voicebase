CREATE TABLE payments.vendor_account ( 
    id                   bigint UNSIGNED NOT NULL,
    entity_id            varchar(150) NOT NULL,
    account_type         varchar(50) NOT NULL,
    access_token         varchar(150) NOT NULL,
    publishable_key      varchar(150) NOT NULL,
    refresh_token        varchar(150) NOT NULL,
    connected_account_id varchar(150) NOT NULL,
    live                 bool NOT NULL,
    scope                varchar(50) NOT NULL,
    lifecycle            varchar(50) NOT NULL,
    change_state         varchar(50) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX      idx_entity_id (entity_id),
    CONSTRAINT pk_vendor_account PRIMARY KEY (id)
) engine=InnoDB;