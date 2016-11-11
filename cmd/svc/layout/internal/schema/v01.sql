CREATE TABLE visit_category (
	id BIGINT UNSIGNED NOT NULL,
	name TEXT NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted TINYINT(1) NOT NULL DEFAULT 0,
	PRIMARY KEY (id)
);

CREATE TABLE visit_layout (
	id BIGINT UNSIGNED NOT NULL,
	name TEXT NOT NULL,
	visit_category_id BIGINT UNSIGNED NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	deleted TINYINT(1) NOT NULL DEFAULT 0,
	CONSTRAINT visit_category_id FOREIGN KEY (visit_category_id) REFERENCES visit_category (id),
	PRIMARY KEY (id)
);

CREATE TABLE visit_layout_version (
	id BIGINT UNSIGNED NOT NULL,
	visit_layout_id BIGINT UNSIGNED NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	saml_location VARCHAR(255) NOT NULL,
	intake_layout_location VARCHAR(255) NOT NULL,
	review_layout_location VARCHAR(255) NOT NULL,
	active TINYINT(1) NOT NULL DEFAULT 0,
	PRIMARY KEY (id),
	CONSTRAINT visit_layout_id FOREIGN KEY (visit_layout_id) REFERENCES visit_layout (id),
	KEY active_saml (visit_layout_id, active)
);

RENAME TABLE visit_category TO visit_category_template;
RENAME TABLE visit_layout TO visit_layout_template;
ALTER TABLE  visit_layout_version DROP CONSTRAINT visit_layout_id;


CREATE TABLE visit_layout (
	id BIGINT UNSIGNED NOT NULL,
	name TEXT NOT NULL,
	visit_category_id BIGINT UNSIGNED NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	deleted TINYINT(1) NOT NULL DEFAULT 0,
	organization_id VARCHAR(64) NOT NULL,
	global TINYINT(1) NOT NULL,
	CONSTRAINT visit_category_id FOREIGN KEY (visit_category_id) REFERENCES visit_category (id),
	PRIMARY KEY (id),
	KEY key_organization_id (organization_id, deleted) 
);

CREATE TABLE visit_category (
	id BIGINT UNSIGNED NOT NULL,
	name TEXT NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_modified TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted TINYINT(1) NOT NULL DEFAULT 0,
	organization_id VARCHAR(64) NOT NULL,
	global TINYINT(1) NOT NULL,
	PRIMARY KEY (id),
	KEY key_organization_id (organization_id, deleted)
);

INSERT INTO visit_category (id, name )
