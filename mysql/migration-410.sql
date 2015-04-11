-- Get rid of unique key and create a non-unique one.. also
-- no need for account_id index
ALTER TABLE auth_token DROP INDEX account_platform;
ALTER TABLE auth_token ADD KEY account_platform (account_id, platform);
ALTER TABLE auth_token DROP INDEX account_id;
