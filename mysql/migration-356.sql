CREATE TABLE admin_dashboard (
    id int(10) unsigned NOT NULL AUTO_INCREMENT,
    name varchar(200) NOT NULL,
    created_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE admin_dashboard_panel (
    id int(10) unsigned NOT NULL AUTO_INCREMENT,
    admin_dashboard_id int(10) unsigned NOT NULL,
    ordinal int unsigned not null,
    columns int unsigned not null,
    type varchar(64) not null,
    config blob not null,
    PRIMARY KEY (id),
    FOREIGN KEY (admin_dashboard_id) REFERENCES admin_dashboard (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
