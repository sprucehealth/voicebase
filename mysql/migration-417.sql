ALTER TABLE account
  ADD COLUMN account_code INT UNSIGNED,
  ADD UNIQUE INDEX(account_code);