-- Drop existing FK's so we can recreeate the primary index
ALTER TABLE account_promotion DROP FOREIGN KEY account_promotion_ibfk_2; 
ALTER TABLE account_promotion DROP FOREIGN KEY account_promotion_ibfk_4; 

-- Drop and recreate the primary index
DROP INDEX `PRIMARY` ON account_promotion;
ALTER TABLE account_promotion ADD id INT NOT NULL AUTO_INCREMENT PRIMARY KEY;

-- Recreate our FKs
ALTER TABLE account_promotion
    ADD CONSTRAINT fk_account_promotion_account_id
        FOREIGN KEY (account_id)
        REFERENCES account (id),
    ADD CONSTRAINT fk_account_promotion_promotion_code_id
        FOREIGN KEY (promotion_code_id)
        REFERENCES promotion_code (id);
