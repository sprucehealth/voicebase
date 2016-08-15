-- ehr_link makes it possible to provide a set of URLs for an entity that links it to a particular EHR
CREATE TABLE ehr_link (
	name VARCHAR(64) NOT NULL,
	entity_id BIGINT UNSIGNED NOT NULL,
	url VARCHAR(255) NOT NULL,
	FOREIGN KEY fk_entity_id (entity_id) REFERENCES entity(id),
	PRIMARY KEY pk_entity_id_name (entity_id, name)
);