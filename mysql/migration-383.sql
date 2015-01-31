-- Make followup its own category
INSERT INTO sku_category (type) VALUES ('followup');

-- INSERT SKUs for all existing pathways for the visit category
INSERT IGNORE INTO sku (type, sku_category_id) 
	SELECT CONCAT(tag, '_visit'), (SELECT ID from sku_category WHERE type = 'visit') 
	FROM clinical_pathway 
	WHERE tag != ('health_condition_acne');

-- INSERT SKUs for all existing pathways for the visit category
INSERT IGNORE INTO sku (type, sku_category_id) 
	SELECT CONCAT(tag, '_followup'), (SELECT ID from sku_category WHERE type = 'followup') 
	FROM clinical_pathway 
	WHERE tag != ('health_condition_acne');
