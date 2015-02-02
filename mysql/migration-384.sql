ALTER TABLE sku MODIFY COLUMN type VARCHAR(128) NOT NULL;

UPDATE sku
	SET type = 'derm_antiaging_and_skin_protection_visit'
	WHERE type like '%derm_antiaging_and_skin_%';
INSERT INTO sku (sku_category_id, type) VALUES ((SELECT id from sku_category where type='followup'), 'derm_antiaging_and_skin_protection_followup');
