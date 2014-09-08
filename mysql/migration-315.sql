CREATE TABLE email_sender (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(64) NOT NULL,
    email VARCHAR(64) NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) character set utf8mb4;

CREATE TABLE email_template (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    type VARCHAR(128) NOT NULL,
    name VARCHAR(200) NOT NULL,
    sender_id INT UNSIGNED NOT NULL,
    subject_template VARCHAR(1024) NOT NULL,
    body_text_template TEXT NOT NULL,
    body_html_template TEXT NOT NULL,
    active BOOL NOT NULL DEFAULT 0,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (sender_id) REFERENCES email_sender (id),
    INDEX (type)
) character set utf8mb4;

INSERT INTO email_sender (name, email) VALUES ('Spruce Support', 'support@sprucehealth.com');

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('unsuitable-for-spruce', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'Patient Visit {{.PatientVisitID}} marked unsuitable for Spruce',
        'Patient Visit {{.PatientVisitID}} marked unsuitable for Spruce', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('medical-record-ready', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'Spruce medical record',
        'We have generated your Spruce medical record which you may download from our website at the following URL. {{.DownloadURL}}', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('passreset-request', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'Reset your Spruce password',
        'We''ve received a request to reset your password. To reset your password click the following link. {{.ResetURL}}', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('passreset-success', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'Reset your Spruce password',
        'You''ve successfully changed your account password.', '', 1);

CREATE UNIQUE INDEX name ON account_available_permission (name);
CREATE UNIQUE INDEX name ON account_group (name);

-- Make sure the superuser group exists. Not sure if there's a way to make INSERT into a noop without the update.
INSERT INTO account_group (name) VALUES ('superuser') ON DUPLICATE KEY UPDATE name = name;

INSERT INTO account_available_permission (name) VALUES ('email.edit'), ('email.view'), ('doctors.edit'), ('doctors.view');

REPLACE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;
