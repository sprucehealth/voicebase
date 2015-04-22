-- Admin site permissions for dynamic config
INSERT INTO account_available_permission (name) VALUES ('cfg.view'), ('cfg.edit');

INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

INSERT INTO account_group (name) VALUES ('cfg');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'cfg'), id
    FROM account_available_permission
    WHERE account_available_permission.name LIKE 'cfg.%';

-- Email campaign tables

CREATE TABLE account_email_sent (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    account_id INT UNSIGNED NOT NULL,
    type VARCHAR(255) NOT NULL,
    time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY account_type (account_id, type),
    CONSTRAINT account_email_sent_account FOREIGN KEY (account_id) REFERENCES account (id)
);

CREATE TABLE account_email_optout (
    account_id INT UNSIGNED NOT NULL,
    type VARCHAR(255) NOT NULL,
    time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, type),
    CONSTRAINT account_email_optout_account FOREIGN KEY (account_id) REFERENCES account (id)
);
