CREATE TABLE `analytics_report` (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    owner_account_id INT UNSIGNED NOT NULL,
    name VARCHAR(200) not null,
    query TEXT not null,
    presentation TEXT not null,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (owner_account_id) REFERENCES account (id)
) character set utf8;

CREATE TABLE `account_group` (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(60) NOT NULL,
    PRIMARY KEY (id)
) character set utf8;

CREATE TABLE `account_group_member` (
    group_id INT UNSIGNED NOT NULL,
    account_id INT UNSIGNED NOT NULL,
    FOREIGN KEY (group_id) REFERENCES account_group (id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES account (id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, group_id),
    KEY group_id (group_id)
) character set utf8;

CREATE TABLE `account_available_permission` (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(60) NOT NULL,
    PRIMARY KEY (id)
) character set utf8;

CREATE TABLE `account_group_permission` (
    group_id INT UNSIGNED NOT NULL,
    permission_id INT UNSIGNED NOT NULL,
    FOREIGN KEY (group_id) REFERENCES account_group (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES account_available_permission (id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, permission_id)
) character set utf8;

INSERT INTO account_available_permission (name) VALUES
    ('admin_accounts.view'),
    ('admin_accounts.edit'),
    ('analytics_reports.view'),
    ('analytics_reports.edit');
