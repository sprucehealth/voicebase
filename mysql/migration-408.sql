-- Adding column to track which skus are still in use
ALTER TABLE sku ADD COLUMN status varchar(32) NOT NULL;

UPDATE sku
SET status='ACTIVE';

-- Inactivating the skus that are no longer in use.
UPDATE sku 
SET status='INACTIVE'
WHERE type in (
	'derm_antiaging_and_skin_protection_followup', 
	'derm_antiaging_and_skin_protection_visit', 
	'derm_hair_loss_in_men_visit', 
	'derm_hair_loss_in_men_followup');