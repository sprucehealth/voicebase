CREATE INDEX account_shadow_expires ON auth_token (account_id, shadow, expires);
ALTER TABLE auth_token DROP INDEX fk_auth_token_account_id;
