
CREATE TABLE form_notify_me (
    email VARCHAR(250) NOT NULL,
    state CHAR(2) NOT NULL,
    platform VARCHAR(128) NOT NULL,
    request_id BIGINT UNSIGNED NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) character set utf8mb4;

CREATE TABLE form_doctor_interest (
    name VARCHAR(250) NOT NULL,
    email VARCHAR(250) NOT NULL,
    states VARCHAR(250) NOT NULL,
    comment VARCHAR(4000) NOT NULL,
    request_id BIGINT UNSIGNED NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) character set utf8mb4;
