CREATE TABLE notification.push_config (
    id                   bigint UNSIGNED NOT NULL,
	external_group_id    varchar(50)  NOT NULL,
	device_token         varbinary(500)  NOT NULL,
	push_endpoint        varchar(300)  NOT NULL,
	platform             varchar(100)  NOT NULL,
	platform_version     varchar(100)  NOT NULL,
	app_version          varchar(100)  NOT NULL,
	device               varchar(100)  NOT NULL,
	device_model         varchar(100)  NOT NULL,
	device_id            varchar(100)  NOT NULL,
	modified             timestamp  NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created              timestamp  NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT pk_external_id PRIMARY KEY (id),
	CONSTRAINT UNIQUE(external_group_id, device_id)
 ) engine=InnoDB;