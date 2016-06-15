-- given that we are looking up verification codes by the token, it is important
-- that they are unique.
ALTER TABLE auth.verification_code ADD UNIQUE KEY (token);

ALTER TABLE auth.verification_code ADD INDEX type_value_lookup (verification_type,verified_value);
