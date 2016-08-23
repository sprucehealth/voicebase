ALTER TABLE saved_queries
	ADD COLUMN title VARCHAR(2048) CHARACTER SET utf8mb4 NOT NULL DEFAULT 'All',
	ADD COLUMN unread INT NOT NULL DEFAULT 0,
	ADD COLUMN total INT NOT NULL DEFAULT 0,
	ADD COLUMN ordinal INT NOT NULL DEFAULT 0;
