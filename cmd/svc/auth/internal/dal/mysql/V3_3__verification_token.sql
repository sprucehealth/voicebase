-- verification_code houses tokens/codes that are time bound and used to verify specific attributes from a user (phone, email, 2fa)
CREATE TABLE auth.verification_code (
    token               varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    code                varchar(12) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    verification_type   varchar(50) NOT NULL,
    verified_value      varchar(255) NOT NULL,
    consumed            bool NOT NULL DEFAULT false,
    created             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires             timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (token, code),
    INDEX idx_expires (expires)
) engine=InnoDB;