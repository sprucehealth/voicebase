-- Track Default Payment Methods
ALTER TABLE payment_method ADD COLUMN is_default bool NOT NULL DEFAULT false;