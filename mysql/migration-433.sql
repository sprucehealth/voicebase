
CREATE TABLE token (
    "purpose" varchar(32) NOT NULL,
    "key" varchar(64) NOT NULL,
    "token" varchar(32) NOT NULL,
    "expires" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY ("purpose", "token"),
    KEY expires ("expires")
);
