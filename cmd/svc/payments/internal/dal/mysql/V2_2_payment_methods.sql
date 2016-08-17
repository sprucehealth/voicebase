ALTER TABLE payments.vendor_account
ADD CONSTRAINT uc_connected_account_id UNIQUE(connected_account_id);

CREATE TABLE payments.customer ( 
    id                   bigint UNSIGNED NOT NULL,
	vendor_account_id    bigint UNSIGNED NOT NULL,
    entity_id            varchar(150) NOT NULL,
    storage_type         varchar(50) NOT NULL, -- This is duplicative with the vendor_account ref but informative
    storage_id           varchar(150) NOT NULL,
	lifecycle            varchar(50) NOT NULL,
    change_state         varchar(50) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX      idx_storage_type (storage_type),
	CONSTRAINT fk_customer_vendor_account_id FOREIGN KEY (vendor_account_id) REFERENCES vendor_account(id) ON DELETE CASCADE, 
    CONSTRAINT uc_entity_id_owning_entity_id UNIQUE(vendor_account_id, entity_id),
	CONSTRAINT uc_storage_id UNIQUE(storage_id),
    CONSTRAINT pk_customer PRIMARY KEY (id)
) engine=InnoDB;

CREATE TABLE payments.payment_method (
	id                   bigint UNSIGNED NOT NULL,
	customer_id          bigint UNSIGNED NOT NULL,
	vendor_account_id    bigint UNSIGNED NOT NULL, -- This is duplicative with the customer_id ref but informative
	entity_id            varchar(150) NOT NULL,
	storage_type         varchar(50) NOT NULL, -- This is duplicative with the vendor_account ref but informative
    storage_id           varchar(150) NOT NULL,
	storage_fingerprint  varchar(150) NOT NULL,
	lifecycle            varchar(50) NOT NULL,
    change_state         varchar(50) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX      idx_entity_id (entity_id),
	INDEX      idx_storage_type (storage_type),
	CONSTRAINT fk_payment_method_customer_id FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE,
	CONSTRAINT fk_payment_method_vendor_account_id FOREIGN KEY (vendor_account_id) REFERENCES vendor_account(id) ON DELETE CASCADE,
	CONSTRAINT uc_storage_id UNIQUE(storage_id),
    CONSTRAINT pk_payment_method PRIMARY KEY (id)
) engine=InnoDB;