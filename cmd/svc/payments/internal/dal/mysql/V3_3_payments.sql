-- Remove the cascading deletes to be safe since we shouldn't be deleting
ALTER TABLE customer DROP FOREIGN KEY fk_customer_vendor_account_id;
ALTER TABLE customer ADD CONSTRAINT fk_customer_vendor_account_id FOREIGN KEY (vendor_account_id) REFERENCES vendor_account(id);
ALTER TABLE payment_method DROP FOREIGN KEY fk_payment_method_customer_id;
ALTER TABLE payment_method ADD CONSTRAINT fk_payment_method_customer_id FOREIGN KEY (customer_id) REFERENCES customer(id);
ALTER TABLE payment_method DROP FOREIGN KEY fk_payment_method_vendor_account_id;
ALTER TABLE payment_method ADD CONSTRAINT fk_payment_method_vendor_account_id FOREIGN KEY (vendor_account_id) REFERENCES vendor_account(id);

CREATE TABLE payments.payment ( 
    id                   bigint UNSIGNED NOT NULL,
    vendor_account_id    bigint UNSIGNED NOT NULL,
    payment_method_id    bigint UNSIGNED, -- Before the payment has been accepted we don't know which
    currency             varchar(50) NOT NULL,
    amount               int UNSIGNED,
    lifecycle            varchar(50) NOT NULL,
    change_state         varchar(50) NOT NULL,
    created              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_payment_vendor_account_id FOREIGN KEY (vendor_account_id) REFERENCES vendor_account(id),
    CONSTRAINT fk_payment_payment_method_id FOREIGN KEY (payment_method_id) REFERENCES payment_method(id), 
    CONSTRAINT pk_payment PRIMARY KEY (id)
) engine=InnoDB;