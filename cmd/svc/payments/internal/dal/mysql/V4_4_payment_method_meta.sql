-- Workers will be polling so index on the lifecycle/change_state
ALTER TABLE payment ADD INDEX idx_lifecycle (lifecycle);
ALTER TABLE payment ADD INDEX idx_change_state (change_state);

ALTER TABLE payment_method 
ADD COLUMN type varchar(50) NOT NULL DEFAULT 'CARD', -- This default will be removed
ADD COLUMN brand varchar(50),
ADD COLUMN last_four varchar(15),
ADD COLUMN exp_month int,
ADD COLUMN exp_year int,
ADD COLUMN tokenization_method varchar(150);

ALTER TABLE payment_method MODIFY COLUMN type varchar(50) NOT NULL;