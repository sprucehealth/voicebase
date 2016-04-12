-- On any interaction with an entity contact, update the modified timestamp on the entity
CREATE TRIGGER ins_entity_contact_entity_modified AFTER INSERT ON entity_contact
    FOR EACH ROW UPDATE entity SET modified = CURRENT_TIMESTAMP WHERE entity.id = NEW.entity_id;

CREATE TRIGGER up_entity_contact_entity_modified AFTER UPDATE ON entity_contact
    FOR EACH ROW UPDATE entity SET modified = CURRENT_TIMESTAMP WHERE entity.id = NEW.entity_id;
	
CREATE TRIGGER del_entity_contact_entity_modified AFTER DELETE ON entity_contact
    FOR EACH ROW UPDATE entity SET modified = CURRENT_TIMESTAMP WHERE entity.id = OLD.entity_id;