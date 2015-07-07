-- Make the address_id column nullable as its not required for a credit
-- card to have an address
ALTER TABLE credit_card DROP FOREIGN KEY credit_card_ibfk_1;
ALTER TABLE credit_card MODIFY COLUMN address_id INT UNSIGNED;
ALTER TABLE credit_card ADD FOREIGN KEY (address_id) REFERENCES address(id);

-- Remove the credit_card_id from the patient_receipt table as that is
-- currently preventing credit cards from being deleted. 
-- Even if we wanted the credit card information we could easily pull
-- that from the stripe charge.
ALTER TABLE patient_receipt DROP FOREIGN KEY patient_receipt_ibfk_2;
ALTER TABLE patient_receipt DROP COLUMN credit_card_id;
