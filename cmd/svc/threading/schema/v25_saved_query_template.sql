-- Indicates whether the saved query is a template or not.
ALTER TABLE saved_queries ADD COLUMN template TINYINT(1) NOT NULL DEFAULT 0;